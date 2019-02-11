package kismetClient

type DataLineReader interface {
	// Returns a function that returns a unique, not previously seen data element.
	// The data element must at least include a latitude, longitude, and ID.
	// The data element may include any kind of extra data that will be stored
	// into a generic interface{} array. The extra data will by typed to a string
	// and stored into the description element of the KML sheet.
	Elements() (func() DataElement, error)
}

// The actual representation of a Kismet Data Element. Designed only to support device data lookups for now.
type DataElement struct {
	// Where ID identifies the data element. This should be unique data
	ID string
	// The latitude coordinate
	Lat float64
	// The longitude coordinate
	Lon float64
	// Extra data that the user throw in
	data []interface{}
	// Flag to tell if there is extra data
	extraData bool
	// Flag to show the fields being set. Used to represent the signal to the caller that there is no more data
	// from repeated calls to the caller
	HasData bool
}

// If the generator that generated this Data Element had extra data either from more queries or otherwise,
// this function will expose that extra data to the caller. The data in the []interface{} will be pointers
// to typed data retrieved directly from the database call. By typed data I mean that it is data that has
// been converted into the go type system.
func (elem *DataElement) GetExtraData() (hasData bool, data []interface{}) {
	hasData	= elem.extraData
	data = elem.data
	return
}

func (elem *DataElement) HasExtraData() bool {
	return elem.extraData
}
