package schema

type DirectiveLocation string

const (
	DirectiveLocationQuery              = "QUERY"
	DirectiveLocationMutation           = "MUTATION"
	DirectiveLocationSubscription       = "SUBSCRIPTION"
	DirectiveLocationField              = "FIELD"
	DirectiveLocationFragmentDefinition = "FRAGMENT_DEFINITION"
	DirectiveLocationFragmentSpread     = "FRAGMENT_SPREAD"
	DirectiveLocationInlineFragment     = "INLINE_FRAGMENT"
)

type DirectiveDefinition struct {
	Name        string
	Description string
	Arguments   map[string]InputValueDefinition
	Locations   []DirectiveLocation
}

type Directive struct {
	Definition *DirectiveDefinition
	Arguments  []*Argument
}
