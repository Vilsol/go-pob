package pob

type TimelessData struct {
	JewelTypeID           string  `xml:"jewelTypeId,attr,omitempty"`
	ConquerorTypeID       string  `xml:"conquerorTypeId,attr,omitempty"`
	DevotionVariant1      string  `xml:"devotionVariant1,attr,omitempty"`
	DevotionVariant2      string  `xml:"devotionVariant2,attr,omitempty"`
	JewelSocketID         string  `xml:"jewelSocketId,attr,omitempty"`
	FallbackWeightModeIdx string  `xml:"fallbackWeightModeIdx,attr,omitempty"`
	SocketFilter          string  `xml:"socketFilter,attr,omitempty"`
	SocketFilterDistance  string  `xml:"socketFilterDistance,attr,omitempty"`
	SocketAllocate        string  `xml:"socketAllocate,attr,omitempty"`
	SearchList            *string `xml:"searchList,attr,omitempty"`
	SearchListFallback    *string `xml:"searchListFallback,attr,omitempty"`
}

func (b *PathOfBuilding) GetTimelessData() TimelessData {
	if b.Build.TimelessData == nil {
		return TimelessData{}
	}
	return *b.Build.TimelessData
}

func (b *PathOfBuilding) SetTimelessData(d TimelessData) {
	b.Build.TimelessData = &d
}
