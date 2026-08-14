package catalog

// fakeDynamo is an in-memory stand-in for DynamoDB that understands exactly
// the access patterns the store uses: get/put/delete by (PK, SK), main-table
// queries of the form `PK = :pk AND begins_with(SK, :sk)`, GSI queries by the
// index's partition attribute with an optional `contains(nameLower, :q)`
// filter, conditional puts/updates inside transactions, and batched deletes.
// It is not a general DynamoDB emulator.

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamo struct {
	mu    sync.Mutex
	items map[string]map[string]ddbtypes.AttributeValue
	// failNext, when set, is returned by the next write call.
	failNext error
}

func newFakeDynamo() *fakeDynamo {
	return &fakeDynamo{items: map[string]map[string]ddbtypes.AttributeValue{}}
}

func s(av ddbtypes.AttributeValue) string {
	if v, ok := av.(*ddbtypes.AttributeValueMemberS); ok {
		return v.Value
	}
	return ""
}

func itemKey(item map[string]ddbtypes.AttributeValue) string {
	return s(item["PK"]) + "|" + s(item["SK"])
}

func keyOf(key map[string]ddbtypes.AttributeValue) string {
	return s(key["PK"]) + "|" + s(key["SK"])
}

func (f *fakeDynamo) GetItem(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &dynamodb.GetItemOutput{Item: f.items[keyOf(in.Key)]}, nil
}

func (f *fakeDynamo) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return nil, err
	}
	f.items[itemKey(in.Item)] = in.Item
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDynamo) DeleteItem(_ context.Context, in *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.items, keyOf(in.Key))
	return &dynamodb.DeleteItemOutput{}, nil
}

func (f *fakeDynamo) UpdateItem(_ context.Context, in *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[keyOf(in.Key)]
	if !ok {
		return nil, &ddbtypes.ConditionalCheckFailedException{Message: aws.String("attribute_exists failed")}
	}
	f.applyUpdate(item, aws.ToString(in.UpdateExpression), in.ExpressionAttributeValues)
	return &dynamodb.UpdateItemOutput{}, nil
}

// applyUpdate handles the two expressions the store issues.
func (f *fakeDynamo) applyUpdate(item map[string]ddbtypes.AttributeValue, expr string, vals map[string]ddbtypes.AttributeValue) {
	switch {
	case strings.HasPrefix(expr, "SET #name"):
		item["name"] = vals[":name"]
		item["releaseDate"] = vals[":rd"]
		item["cardTotal"] = vals[":ct"]
		item["logoUrl"] = vals[":lu"]
		item["updatedAt"] = vals[":ua"]
	case strings.HasPrefix(expr, "ADD cardCount"):
		delta, _ := strconv.Atoi(vals[":d"].(*ddbtypes.AttributeValueMemberN).Value)
		current := 0
		if n, ok := item["cardCount"].(*ddbtypes.AttributeValueMemberN); ok {
			current, _ = strconv.Atoi(n.Value)
		}
		item["cardCount"] = &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(current + delta)}
	default:
		panic("fakeDynamo: unrecognized update expression: " + expr)
	}
}

func (f *fakeDynamo) Query(_ context.Context, in *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []map[string]ddbtypes.AttributeValue
	pk := s(in.ExpressionAttributeValues[":pk"])

	if in.IndexName == nil {
		prefix := s(in.ExpressionAttributeValues[":sk"])
		for _, item := range f.items {
			if s(item["PK"]) == pk && strings.HasPrefix(s(item["SK"]), prefix) {
				out = append(out, item)
			}
		}
	} else {
		attr := map[string]string{gsi1: "GSI1PK", gsi2: "GSI2PK", gsi3: "GSI3PK"}[*in.IndexName]
		for _, item := range f.items {
			if s(item[attr]) != pk {
				continue
			}
			if in.FilterExpression != nil {
				switch *in.FilterExpression {
				case "contains(nameLower, :q)":
					if !strings.Contains(s(item["nameLower"]), s(in.ExpressionAttributeValues[":q"])) {
						continue
					}
				case "attribute_exists(tcgplayerId)":
					if _, ok := item["tcgplayerId"]; !ok {
						continue
					}
				default:
					panic("fakeDynamo: unrecognized filter expression: " + *in.FilterExpression)
				}
			}
			out = append(out, item)
		}
		if *in.IndexName == gsi1 {
			sortByAttr(out, "GSI1SK")
		}
	}
	if in.Limit != nil && len(out) > int(*in.Limit) {
		out = out[:int(*in.Limit)]
	}
	return &dynamodb.QueryOutput{Items: out}, nil
}

func sortByAttr(items []map[string]ddbtypes.AttributeValue, attr string) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && s(items[j][attr]) < s(items[j-1][attr]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func (f *fakeDynamo) BatchWriteItem(_ context.Context, in *dynamodb.BatchWriteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, reqs := range in.RequestItems {
		if len(reqs) > 25 {
			panic("fakeDynamo: BatchWriteItem over the 25-item ceiling")
		}
		for _, r := range reqs {
			if r.DeleteRequest != nil {
				delete(f.items, keyOf(r.DeleteRequest.Key))
			}
		}
	}
	return &dynamodb.BatchWriteItemOutput{}, nil
}

func (f *fakeDynamo) TransactWriteItems(_ context.Context, in *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return nil, err
	}
	// Validate all conditions first — a real transaction is all-or-nothing.
	cancelled := false
	for _, t := range in.TransactItems {
		switch {
		case t.Put != nil && t.Put.ConditionExpression != nil:
			if _, exists := f.items[itemKey(t.Put.Item)]; exists &&
				*t.Put.ConditionExpression == "attribute_not_exists(PK)" {
				cancelled = true
			}
		case t.Update != nil:
			if _, exists := f.items[keyOf(t.Update.Key)]; !exists {
				cancelled = true
			}
		}
	}
	if cancelled {
		return nil, &ddbtypes.TransactionCanceledException{
			CancellationReasons: []ddbtypes.CancellationReason{
				{Code: aws.String("ConditionalCheckFailed")},
			},
		}
	}
	for _, t := range in.TransactItems {
		switch {
		case t.Put != nil:
			f.items[itemKey(t.Put.Item)] = t.Put.Item
		case t.Delete != nil:
			delete(f.items, keyOf(t.Delete.Key))
		case t.Update != nil:
			f.applyUpdate(f.items[keyOf(t.Update.Key)],
				aws.ToString(t.Update.UpdateExpression), t.Update.ExpressionAttributeValues)
		}
	}
	return &dynamodb.TransactWriteItemsOutput{}, nil
}
