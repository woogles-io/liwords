package memento

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

func TestRenderSlotOneAtATime(t *testing.T) {
	release, err := acquireRenderSlot(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// A second render waits, then gives up.
	start := time.Now()
	_, err = acquireRenderSlot(context.Background(), 50*time.Millisecond)
	if !errors.Is(err, errRenderBusy) {
		t.Fatalf("got %v, want errRenderBusy", err)
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Fatal("gave up before the wait was over")
	}

	// A request that goes away stops waiting at once.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = acquireRenderSlot(ctx, time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}

	// Once the first render finishes, the next one gets the slot.
	release()
	release2, err := acquireRenderSlot(context.Background(), 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	release2()
}

func TestRenderSlotWaiterGetsFreedSlot(t *testing.T) {
	release, err := acquireRenderSlot(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		release()
	}()
	release2, err := acquireRenderSlot(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	release2()
}

func TestRenderStopsWhenCanceled(t *testing.T) {
	gh, err := os.ReadFile("./testdata/gh1.json")
	if err != nil {
		t.Fatal(err)
	}
	hist := &macondopb.GameHistory{}
	if err := json.Unmarshal(gh, hist); err != nil {
		t.Fatal(err)
	}
	for _, ver := range []int{0, 2} {
		wf := WhichFile{FileType: "animated-gif", Version: ver, WhichColor: -1}

		if _, err := RenderImage(context.Background(), hist, wf); err != nil {
			t.Fatalf("version %d: %v", ver, err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		start := time.Now()
		_, err := RenderImage(ctx, hist, wf)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("version %d: got %v, want context.Canceled", ver, err)
		}
		if d := time.Since(start); d > 100*time.Millisecond {
			t.Fatalf("version %d: canceled render took %v", ver, d)
		}
	}
}
