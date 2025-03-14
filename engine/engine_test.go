package engine

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/bombsimon/logrusr/v3"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

type testConditional struct {
	err   error
	ret   bool
	sleep bool
}

func (t testConditional) Evaluate(ctx context.Context, log logr.Logger, condCtx ConditionContext) (ConditionResponse, error) {
	if t.sleep {
		time.Sleep(5 * time.Second)
	}
	if t.ret {
		return ConditionResponse{Matched: t.ret, Incidents: []IncidentContext{{FileURI: "test"}}}, t.err
	} else {
		return ConditionResponse{Matched: t.ret}, t.err
	}
}

func (t testConditional) Ignorable() bool {
	return true
}

func createTestConditional(b bool, e error, sleep bool) Conditional {
	return &testConditional{
		err:   e,
		ret:   b,
		sleep: sleep,
	}
}

type testChainableConditionalAs struct {
	err           error
	documentedKey string
	AsValue       interface{}
}

func (t testChainableConditionalAs) Evaluate(ctx context.Context, log logr.Logger, condCtx ConditionContext) (ConditionResponse, error) {
	return ConditionResponse{
		Matched: true,
		TemplateContext: map[string]interface{}{
			t.documentedKey: t.AsValue,
		},
		Incidents: []IncidentContext{{}},
	}, t.err
}

func (t testChainableConditionalAs) Ignorable() bool {
	return true
}

type testChainableConditionalFrom struct {
	FromName      string
	DocumentedKey string
	FromValue     interface{}
}

func (t testChainableConditionalFrom) Ignorable() bool {
	return true
}

func (t testChainableConditionalFrom) Evaluate(ctx context.Context, log logr.Logger, condCtx ConditionContext) (ConditionResponse, error) {
	if v, ok := condCtx.Template[t.FromName]; ok {
		if reflect.DeepEqual(v.Extras[t.DocumentedKey], t.FromValue) {
			return ConditionResponse{
				Matched:         true,
				TemplateContext: map[string]interface{}{},
			}, nil
		}
	}
	return ConditionResponse{}, fmt.Errorf("unable to find from in context")
}

func TestEvaluateAndConditions(t *testing.T) {

	testCases := []struct {
		Name       string
		Conditions []ConditionEntry
		IsError    bool
		IsMatched  bool
	}{
		{
			Name: "Base Case",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
			},
			IsMatched: false,
		},
		{
			Name: "And two inner conditions",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
			},
			IsMatched: false,
		},
		{
			Name: "And two inner conditions failure",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
				{
					ProviderSpecificConfig: createTestConditional(true, nil, false),
				},
			},
		},
		{
			Name: "And two conditions with nested conditions",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: AndCondition{Conditions: []ConditionEntry{
						{
							From:                   "",
							As:                     "",
							ProviderSpecificConfig: createTestConditional(false, nil, false),
						},
						{
							From:                   "",
							As:                     "",
							ProviderSpecificConfig: createTestConditional(false, nil, false),
						},
					}},
				},
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
			},
			IsMatched: false,
		},
		{
			Name: "And two conditions with nested conditions failure",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: AndCondition{
						Conditions: []ConditionEntry{
							{
								ProviderSpecificConfig: createTestConditional(true, nil, false),
							},
							{
								ProviderSpecificConfig: createTestConditional(false, nil, false),
							},
						},
					},
				},
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
			},
		},
	}
	testString := "testing"
	logrusLog := logrus.New()
	log := logrusr.New(logrusLog)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			rule := Rule{
				Perform: Perform{
					Message: Message{
						Text: &testString,
					},
				},
				When: AndCondition{Conditions: tc.Conditions},
			}

			ret, err := processRule(context.TODO(), rule, ConditionContext{
				Template: make(map[string]ChainTemplate),
			}, log)
			if err != nil && !tc.IsError {
				t.Errorf("got err: %v, expected no error", err)
			}
			if ret.Matched != tc.IsMatched {
				t.Errorf("Expected to be: %v, but got: %v", tc.IsMatched, ret)
			}
		})
	}

}

func TestEvaluateOrConditions(t *testing.T) {

	testCases := []struct {
		Name       string
		Conditions []ConditionEntry
		IsError    bool
		IsMatched  bool
	}{
		{
			Name: "Base Case",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
			},
			IsMatched: false,
		},
		{
			Name: "or two inner conditions",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: createTestConditional(true, nil, false),
				},
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
			},
			IsMatched: true,
		},
		{
			Name: "or two inner conditions false first",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
				{
					ProviderSpecificConfig: createTestConditional(true, nil, false),
				},
			},
			IsMatched: true,
		},
		{
			Name: "or two inner conditions failure",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: createTestConditional(true, nil, false),
				},
				{
					ProviderSpecificConfig: createTestConditional(true, nil, false),
				},
			},
			IsMatched: true,
		},
		{
			Name: "And two conditions with nested conditions",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: OrCondition{Conditions: []ConditionEntry{
						{
							ProviderSpecificConfig: createTestConditional(false, nil, false),
						},
						{
							ProviderSpecificConfig: createTestConditional(true, nil, false),
						},
					}},
				},
				{
					ProviderSpecificConfig: createTestConditional(true, nil, false),
				},
			},
			IsMatched: true,
		},
		{
			Name: "or two conditions with nested conditions failure",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: OrCondition{
						Conditions: []ConditionEntry{
							{
								ProviderSpecificConfig: createTestConditional(false, nil, false),
							},
							{

								ProviderSpecificConfig: createTestConditional(false, nil, false),
							},
						},
					},
				},
				{
					ProviderSpecificConfig: createTestConditional(false, nil, false),
				},
			},
		},
	}
	testString := "testing"
	logrusLog := logrus.New()
	log := logrusr.New(logrusLog)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			rule := Rule{
				Perform: Perform{
					Message: Message{
						Text: &testString,
					},
				},
				When: OrCondition{tc.Conditions},
			}
			ret, err := processRule(context.TODO(), rule, ConditionContext{
				Template: make(map[string]ChainTemplate),
			}, log)
			if err != nil && !tc.IsError {
				t.Errorf("got err: %v, expected no error", err)
			}
			if ret.Matched != tc.IsMatched {
				t.Errorf("Expected to be: %v, but got: %v", tc.IsMatched, ret)
			}
		})
	}
}

func TestChainConditions(t *testing.T) {

	testCases := []struct {
		Name       string
		Conditions []ConditionEntry
		IsError    bool
		IsMatched  bool
	}{
		{
			Name: "Test Basic single chain",
			Conditions: []ConditionEntry{
				{
					As: "testing",
					ProviderSpecificConfig: testChainableConditionalAs{
						documentedKey: "filepaths",
						AsValue:       []string{"test.yaml"},
					},
				},
				{
					From: "testing",
					ProviderSpecificConfig: testChainableConditionalFrom{
						FromName:      "testing",
						DocumentedKey: "filepaths",
						FromValue:     []string{"test.yaml"},
					},
				},
			},
			IsMatched: true,
		},
		{
			Name: "Test or chain As provided by one element in or block",
			Conditions: []ConditionEntry{
				{
					As: "testing",
					ProviderSpecificConfig: OrCondition{
						Conditions: []ConditionEntry{
							{
								ProviderSpecificConfig: testChainableConditionalAs{
									documentedKey: "filepaths",
									AsValue:       []string{"test.yaml"},
								},
							},
							{
								ProviderSpecificConfig: createTestConditional(true, nil, false),
							},
						},
					},
				},
				{
					From: "testing",
					ProviderSpecificConfig: testChainableConditionalFrom{
						FromName:      "testing",
						DocumentedKey: "filepaths",
						FromValue:     []string{"test.yaml"},
					},
				},
			},
			IsMatched: true,
		},
		{
			Name: "Test and chain As provided and block",
			Conditions: []ConditionEntry{
				{
					As: "testing",
					ProviderSpecificConfig: AndCondition{
						Conditions: []ConditionEntry{
							{
								ProviderSpecificConfig: testChainableConditionalAs{
									documentedKey: "filepaths",
									AsValue:       []string{"test.yaml"},
								},
							},
							{
								ProviderSpecificConfig: createTestConditional(true, nil, false),
							},
						},
					},
				},
				{
					From: "testing",
					ProviderSpecificConfig: testChainableConditionalFrom{
						FromName:      "testing",
						DocumentedKey: "filepaths",
						FromValue:     []string{"test.yaml"},
					},
				},
			},
			IsMatched: true,
		},
		{
			Name: "Test and chain As provided by one element in as block",
			Conditions: []ConditionEntry{
				{
					ProviderSpecificConfig: AndCondition{
						Conditions: []ConditionEntry{
							{
								As: "testing",
								ProviderSpecificConfig: OrCondition{
									Conditions: []ConditionEntry{
										{
											As: "testing",
											ProviderSpecificConfig: testChainableConditionalAs{
												documentedKey: "filepaths",
												AsValue:       []string{"test.yaml"},
											},
										},
									},
								},
							},
							{
								ProviderSpecificConfig: createTestConditional(true, nil, false),
							},
						},
					},
				},
				{
					From: "testing",
					ProviderSpecificConfig: testChainableConditionalFrom{
						FromName:      "testing",
						DocumentedKey: "filepaths",
						FromValue:     []string{"test.yaml"},
					},
				},
			},
			IsMatched: true,
		},
	}
	testString := "testing"
	logrusLog := logrus.New()
	log := logrusr.New(logrusLog)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			rule := Rule{
				Perform: Perform{
					Message: Message{
						Text: &testString,
					},
				},
				When: OrCondition{tc.Conditions},
			}
			ret, err := processRule(context.TODO(), rule, ConditionContext{
				Template: make(map[string]ChainTemplate),
			}, log)
			if err != nil && !tc.IsError {
				t.Errorf("got err: %v, expected no error", err)
			}
			if ret.Matched != tc.IsMatched {
				t.Errorf("Expected to be: %v, but got: %v", tc.IsMatched, ret)
			}
		})
	}
}

func TestRuleEngine(t *testing.T) {
	woo := "WOO"
	wooFalse := "WOO - False"
	testCases := []struct {
		Name  string
		Rules []RuleSet
	}{
		{
			Name: "Test Running",
			Rules: []RuleSet{
				{
					Rules: []Rule{
						{
							Perform: Perform{Message: Message{Text: &woo}},
							When:    createTestConditional(false, nil, true),
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, true),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, true),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, true),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, true),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "test 11 rules",
			Rules: []RuleSet{
				{
					Rules: []Rule{
						{
							Perform: Perform{Message: Message{Text: &woo}},
							When:    createTestConditional(false, nil, false),
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
						{
							Perform: Perform{Message: Message{Text: &wooFalse}},
							When: AndCondition{
								Conditions: []ConditionEntry{
									{
										ProviderSpecificConfig: createTestConditional(false, nil, false),
									},
									{
										ProviderSpecificConfig: createTestConditional(true, nil, false),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	logrusLog := logrus.New()
	log := logrusr.New(logrusLog)
	ruleEngine := CreateRuleEngine(context.Background(), 10, log)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			start := time.Now()
			ruleEngine.RunRules(context.Background(), tc.Rules)
			// make sure that the the test takes only as long as we would expect w/ the sleeps
			if time.Since(start) >= 11*time.Second {
				t.Fail()
			}

		})
	}
}

func Test_parseTagsFromPerformString(t *testing.T) {
	tests := []struct {
		name      string
		tagString string
		want      []string
		wantErr   bool
	}{
		{
			name:      "tc1",
			tagString: "test1,test2,test3,test4",
			want:      []string{"test1", "test2", "test3", "test4"},
		},
		{
			name:      "tc2",
			tagString: "test1-tag,",
			want:      []string{"test1-tag"},
		},
		{
			name:      "tc3",
			tagString: "test1",
			want:      []string{"test1"},
		},
		{
			name:      "tc4",
			tagString: "Category=test1,test2,test3,test4",
			want:      []string{"test1", "test2", "test3", "test4"},
		},
		{
			name:      "tc5",
			tagString: "Category=test1,",
			want:      []string{"test1"},
		},
		{
			name:      "tc6",
			tagString: "test1, test2, test3, test4",
			want:      []string{"test1", "test2", "test3", "test4"},
		},
		{
			name:      "tc6",
			tagString: "test tag1, test tag2",
			want:      []string{"test tag1", "test tag2"},
		},
		{
			name:      "tc7",
			tagString: "Category==test1,test2,test3,test4",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "tc8 - spaces in the tag values",
			tagString: "Category 1=test 1,test 2,test 3,test 4",
			want:      []string{"test 1", "test 2", "test 3", "test 4"},
			wantErr:   false,
		},
		{
			name:      "tc9 - parentheses in the tag values",
			tagString: "Category (1)=test (1),test (2),test (3),",
			want:      []string{"test (1)", "test (2)", "test (3)"},
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTagsFromPerformString(tt.tagString)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTagsFromPerformString() = %v, want %v", got, tt.want)
			}
			if tt.wantErr != (err != nil) {
				t.Errorf("parseTagsFromPerformString() = error %v, want %v", err != nil, tt.wantErr)
			}
		})
	}
}
