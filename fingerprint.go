package config

var _hashCode string

// NewHashCode if the _hashCode variable is empty, then this function will populate it with the passed in param and return it.
func NewHashCode(hashCode string) string {
	if _hashCode == "" {
		_hashCode = hashCode
	}
	return _hashCode
}

// HashCode gets the existing value stored in _hashCode
func HashCode() string {
	return _hashCode
}
