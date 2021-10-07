package tree

import (
	"redwood.dev/tree/pb"
	"redwood.dev/types"
)

var (
	GenesisTxID = types.IDFromString("genesis")
	EmptyHash   = types.Hash{}
)

type Tx = pb.Tx
type Patch = pb.Patch
type TxStatus = pb.TxStatus

var (
	TxStatusUnknown   = pb.TxStatusUnknown
	TxStatusInMempool = pb.TxStatusInMempool
	TxStatusInvalid   = pb.TxStatusInvalid
	TxStatusValid     = pb.TxStatusValid

	ParsePatch     = pb.ParsePatch
	ParsePatchPath = pb.ParsePatchPath
)
