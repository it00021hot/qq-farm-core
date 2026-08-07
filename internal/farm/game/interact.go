package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/interactpb"
)

// interactRecordsCandidates mirrors Node interact.ts RPC fallbacks.
var interactRecordsCandidates = [][2]string{
	{interactService, "InteractRecords"},
	{interactService, "GetInteractRecords"},
	{interactVisitorService, "InteractRecords"},
	{interactVisitorService, "GetInteractRecords"},
}

func (a *API) sendInteract(ctx context.Context, service, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, service, method, nonNilBody(body))
	return raw, err
}

// InteractRecords fetches visitor interaction records (with Node-style service/method fallbacks).
func (a *API) InteractRecords(ctx context.Context) (*interactpb.InteractRecordsReply, error) {
	body := marshalMessage(&interactpb.InteractRecordsRequest{})
	var errs []string
	for _, cand := range interactRecordsCandidates {
		raw, err := a.sendInteract(ctx, cand[0], cand[1], body)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s.%s: %v", cand[0], cand[1], err))
			continue
		}
		reply := &interactpb.InteractRecordsReply{}
		if err := unmarshalMessage(raw, reply); err != nil {
			errs = append(errs, fmt.Sprintf("%s.%s decode: %v", cand[0], cand[1], err))
			continue
		}
		return reply, nil
	}
	if len(errs) == 0 {
		return nil, fmt.Errorf("interact records: no candidates")
	}
	return nil, fmt.Errorf("interact records failed: %s", strings.Join(errs, " | "))
}

// GetInteractInfo fetches interaction info entries.
func (a *API) GetInteractInfo(ctx context.Context) (*interactpb.GetInteractInfoReply, error) {
	raw, err := a.sendInteract(ctx, interactService, "GetInteractInfo", marshalMessage(&interactpb.GetInteractInfoRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &interactpb.GetInteractInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetInteractSummary fetches interaction summary stats.
func (a *API) GetInteractSummary(ctx context.Context) (*interactpb.GetInteractSummaryReply, error) {
	raw, err := a.sendInteract(ctx, interactService, "GetInteractSummary", marshalMessage(&interactpb.GetInteractSummaryRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &interactpb.GetInteractSummaryReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// DismissInteractPopup dismisses the interact popup.
func (a *API) DismissInteractPopup(ctx context.Context) (*interactpb.DismissInteractPopupReply, error) {
	raw, err := a.sendInteract(ctx, interactService, "DismissInteractPopup", marshalMessage(&interactpb.DismissInteractPopupRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &interactpb.DismissInteractPopupReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
