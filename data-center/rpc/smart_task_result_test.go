package rpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"micro_service/services"
	"testing"
)

func TestSmartTaskResolutionRequiresConcreteFinancialResultSource(t *testing.T) {
	client, _ := gogameRPCFixture(t)
	request := &services.GogameStorageRequest{Operation: "smartTaskResolve", Payload: `{"agentId":7,"userId":42,"roomType":58,"taskId":"original"}`}
	// The generic repository's placeholder response must never be treated as a
	// resolved financial result; this operation uses the concrete DAO and ES.
	if _, err := client.Call(gogameRPCContext(), request); status.Code(err) != codes.Unavailable {
		t.Fatal(err)
	}
}
