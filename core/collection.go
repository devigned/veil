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

// CollectionNamedSlice is a wrapper type for []NamedType
type CollectionNamedSlice []NamedStruct

// Enumerate will create an enumerator for a []NamedType
func (nt CollectionNamedSlice) Enumerate() collection.Enumerator {
	var interfaceSlice = make([]interface{}, len(nt))
	for i, d := range nt {
		interfaceSlice[i] = d
	}
	return collection.AsEnumerator(interfaceSlice...)
}
