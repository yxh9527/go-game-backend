package rpc

import (
	"context"
	"encoding/json"
	"io"
	"micro_service/services"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type gogameRepositoryFixture struct {
	mu               sync.Mutex
	row, state       map[string]any
	commit, rollback int
	released         chan struct{}
}

func (f *gogameRepositoryFixture) GogameCall(context.Context, string, string) (any, error) {
	return []map[string]any{{"id": 1}}, nil
}
func (f *gogameRepositoryFixture) WithGogameMatch(ctx context.Context, id int64, work func(map[string]any, map[string]any) error) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	defer func() {
		if f.released != nil {
			close(f.released)
			f.released = nil
		}
	}()
	if id != 9 {
		return false, nil
	}
	row := map[string]any{"id": 9, "agent_id": 11}
	state := map[string]any{"registrations": map[string]any{}, "outbox": []any{}}
	if err := work(row, state); err != nil {
		f.rollback++
		return true, err
	}
	f.row, f.state = row, state
	f.commit++
	return true, nil
}
func gogameRPCFixture(t *testing.T) (services.GogameStorageServiceClient, *gogameRepositoryFixture) {
	t.Helper()
	t.Setenv("GOGAME_DATA_CENTER_TOKEN", "test-storage-token")
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	repo := &gogameRepositoryFixture{}
	services.RegisterGogameStorageServiceServer(server, newGogameStorageService(repo))
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	connection, err := grpc.NewClient("passthrough:///storage-test", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { connection.Close() })
	return services.NewGogameStorageServiceClient(connection), repo
}
func gogameRPCContext() context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-gogame-storage-token", "test-storage-token"))
}

func TestGogameRPCRequiresConfiguredServiceCredential(t *testing.T) {
	client, _ := gogameRPCFixture(t)
	if _, err := client.Call(context.Background(), &services.GogameStorageRequest{Operation: "listRooms", Payload: "{}"}); status.Code(err) != codes.PermissionDenied {
		t.Fatal(err)
	}
	reply, err := client.Call(gogameRPCContext(), &services.GogameStorageRequest{Operation: "listRooms", Payload: "{}"})
	if err != nil || reply.Phase != "result" {
		t.Fatal(reply, err)
	}
	t.Setenv("GOGAME_DATA_CENTER_TOKEN", "")
	if _, err := client.Call(gogameRPCContext(), &services.GogameStorageRequest{Operation: "listRooms", Payload: "{}"}); status.Code(err) != codes.Unavailable {
		t.Fatal(err)
	}
}

func TestGogameRPCConfirmsCommitOnlyAfterRepositorySuccess(t *testing.T) {
	client, repo := gogameRPCFixture(t)
	stream, err := client.MatchTransaction(gogameRPCContext())
	if err != nil {
		t.Fatal(err)
	}
	if err = stream.Send(&services.GogameStorageRequest{Operation: "begin", Payload: `{"id":9}`}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := stream.Recv()
	if err != nil || snapshot.Phase != "snapshot" {
		t.Fatal(snapshot, err)
	}
	var doc map[string]any
	if err = json.Unmarshal([]byte(snapshot.Payload), &doc); err != nil {
		t.Fatal(err)
	}
	doc["row"].(map[string]any)["state"] = 4
	next, _ := json.Marshal(doc)
	if err = stream.Send(&services.GogameStorageRequest{Operation: "commit", Payload: string(next)}); err != nil {
		t.Fatal(err)
	}
	stream.CloseSend()
	committed, err := stream.Recv()
	if err != nil || committed.Phase != "committed" {
		t.Fatal(committed, err)
	}
	if _, err = stream.Recv(); err != io.EOF {
		t.Fatal(err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.commit != 1 || repo.rollback != 0 {
		t.Fatalf("bad commit count %d %d", repo.commit, repo.rollback)
	}
}

func TestGogameRPCDisconnectRollsBackAndReleasesTransaction(t *testing.T) {
	client, repo := gogameRPCFixture(t)
	released := make(chan struct{})
	repo.released = released
	ctx, cancel := context.WithCancel(gogameRPCContext())
	stream, err := client.MatchTransaction(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stream.Send(&services.GogameStorageRequest{Operation: "begin", Payload: `{"id":9}`})
	snapshot, err := stream.Recv()
	if err != nil || snapshot.Phase != "snapshot" {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("disconnect did not release transaction")
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.commit != 0 || repo.rollback != 1 {
		t.Fatal("cancelled transaction committed")
	}
}
