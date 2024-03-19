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

func (s FeatureSet) Union(other FeatureSet) FeatureSet {
	fs := make(FeatureSet, len(s)+len(other))
	for feature := range s {
		fs[feature] = struct{}{}
	}
	for feature := range other {
		fs[feature] = struct{}{}
	}
	return fs
}
