// +build !otto

package tree

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
	"rogchap.com/v8go"

	//"github.com/saibing/go-v8"

	"redwood.dev/blob"
	"redwood.dev/log"
	"redwood.dev/state"
	"redwood.dev/tree/nelson"
	"redwood.dev/types"
	"redwood.dev/utils"
)

type jsResolver struct {
	log.Logger
	vm            *v8go.Context
	internalState map[string]interface{}
}

// Ensure jsResolver conforms to the Resolver interface
var _ Resolver = (*jsResolver)(nil)

func NewJSResolver(config state.Node, internalState map[string]interface{}) (_ Resolver, err error) {
	defer utils.Annotate(&err, "NewJSResolver")

	// srcval, exists, err := nelson.GetValueRecursive(config, state.Keypath("src"), nil)
	srcval, exists, err := config.Value(state.Keypath("src"), nil)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.Errorf("js resolver needs a 'src' param")
	}

	readableSrc, ok := nelson.GetReadCloser(srcval)
	if !ok {
		return nil, errors.Errorf("js resolver needs a 'src' param of type string, []byte, or io.ReadCloser (got %T)", srcval)
	}
	defer readableSrc.Close()

	srcStr, err := ioutil.ReadAll(readableSrc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	v8ctx, _ := v8go.NewContext(nil)

	_, err = v8ctx.RunScript("var global = {}; var newStateJSON; "+string(srcStr), "")
	if err != nil {
		return nil, err
	}

	internalStateBytes, err := json.Marshal(internalState)
	if err != nil {
		return nil, err
	}

	internalStateScript := "global.init(" + string(internalStateBytes) + ")"
	_, err = v8ctx.RunScript(internalStateScript, "")
	if err != nil {
		return nil, err
	}

	return &jsResolver{Logger: log.NewLogger("resolver:js"), vm: v8ctx, internalState: internalState}, nil
}

func (r *jsResolver) InternalState() map[string]interface{} {
	return r.internalState
}

func (r *jsResolver) ResolveState(node state.Node, blobStore blob.Store, sender types.Address, txID types.ID, parents []types.ID, patches []Patch) (err error) {
	defer utils.Annotate(&err, "jsResolver.ResolveState")

	convertedPatches := make([]interface{}, len(patches))
	for i, patch := range patches {
		val, err := patch.Value()
		if err != nil {
			return err
		}

		convertedPatch := map[string]interface{}{
			"keys": patch.Keypath.PartStrings(),
			"val":  val,
		}

		if patch.Range != nil {
			convertedPatch["range"] = []interface{}{patch.Range.Start, patch.Range.End}
		}
		convertedPatches[i] = convertedPatch
	}

	stateJSON, err := json.Marshal(node)
	if err != nil {
		return err
	}

	var parentsArr []string
	for i := range parents {
		parentsArr = append(parentsArr, parents[i].String())
	}
	parentsArrJSON, _ := json.Marshal(parentsArr)
	convertedPatchesJSON, _ := json.Marshal(convertedPatches)

	script := "newStateJSON = global.resolve_state(" + string(stateJSON) + ", '" + sender.String() + "', '" + txID.String() + "', " + string(parentsArrJSON) + ", " + string(convertedPatchesJSON) + ")"
	_, err = r.vm.RunScript(script, "")
	if err != nil {
		return err
	}

	newStateJSONVal, err := r.vm.RunScript("newStateJSON", "")
	if err != nil {
		return err
	}

	var output map[string]interface{}
	err = json.Unmarshal([]byte(newStateJSONVal.String()), &output)
	if err != nil {
		return err
	}

	r.internalState = output["internalState"].(map[string]interface{})
	node.Set(nil, nil, output["state"])
	return nil
}
