package tokenizer

import "testing"

func TestTokenizePreservesTechnicalTerms(t *testing.T) {
	terms := Tokenize("AWS::SQS::Queue visibilityTimeout eksctl")
	want := map[string]bool{"aws::sqs::queue": true, "visibilitytimeout": true, "eksctl": true}
	for _, term := range terms {
		delete(want, term)
	}
	if len(want) != 0 {
		t.Fatalf("missing terms: %v (got %v)", want, terms)
	}
}

func TestTokenizeWithFreq(t *testing.T) {
	freq := TokenizeWithFreq("SQS SQS queue")
	if freq["sqs"] != 2 {
		t.Fatalf("sqs freq=%d", freq["sqs"])
	}
	if freq["queue"] != 1 {
		t.Fatalf("queue freq=%d", freq["queue"])
	}
}
