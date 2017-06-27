package core

import "github.com/marstr/collection"

// CollectionStringSlice is a wrapper type for []string
type CollectionStringSlice []string

// Enumerate will create an enumerator for a []string
func (sl CollectionStringSlice) Enumerate() collection.Enumerator {
	var interfaceSlice = make([]interface{}, len(sl))
	for i, d := range sl {
		interfaceSlice[i] = d
	}
	return collection.AsEnumerator(interfaceSlice...)
}
