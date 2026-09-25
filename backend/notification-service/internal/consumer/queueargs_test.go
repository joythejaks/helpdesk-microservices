package consumer

import "testing"

func TestMainQueueArgs_ClassicKeepsThePreQuorumArguments(t *testing.T) {
	args := mainQueueArgs(QueueTypeClassic)
	if len(args) != 1 || args["x-dead-letter-exchange"] != dlxExchange {
		t.Fatalf("classic must declare exactly the pre-quorum arguments, got %v", args)
	}
	if _, ok := args["x-queue-type"]; ok {
		t.Fatal("classic must not set x-queue-type (would differ from existing queues)")
	}
}

func TestMainQueueArgs_QuorumAddsQueueType(t *testing.T) {
	args := mainQueueArgs(QueueTypeQuorum)
	if args["x-queue-type"] != QueueTypeQuorum || args["x-dead-letter-exchange"] != dlxExchange {
		t.Fatalf("quorum must keep the DLX and add x-queue-type, got %v", args)
	}
}

func TestDLQArgs(t *testing.T) {
	if dlqArgs(QueueTypeClassic) != nil {
		t.Fatal("classic DLQ keeps nil arguments")
	}
	if dlqArgs(QueueTypeQuorum)["x-queue-type"] != QueueTypeQuorum {
		t.Fatal("quorum DLQ must be a quorum queue")
	}
}

func TestValidateQueueType(t *testing.T) {
	for _, ok := range []string{"classic", "quorum"} {
		if err := ValidateQueueType(ok); err != nil {
			t.Fatalf("%q should be valid: %v", ok, err)
		}
	}
	if ValidateQueueType("quorom") == nil || ValidateQueueType("") == nil {
		t.Fatal("typos and empty values must be rejected")
	}
}
