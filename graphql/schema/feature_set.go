package schema

type FeatureSet map[string]struct{}

func NewFeatureSet(features ...string) FeatureSet {
	fs := make(FeatureSet, len(features))
	for _, feature := range features {
		fs[feature] = struct{}{}
	}
	return fs
}

func (s FeatureSet) Has(feature string) bool {
	_, ok := s[feature]
	return ok
}

func (s FeatureSet) IsSubsetOf(other FeatureSet) bool {
	for feature := range s {
		if _, ok := other[feature]; !ok {
			return false
		}
	}
	return true
}
