package services

type ICollectReturn interface {
	HandleCollectReturn(data string) error
}

var localCollectReturn ICollectReturn

func CollectReturn() ICollectReturn {
	if localCollectReturn == nil {
		panic("impl not found for ICollectReturn")
	}
	return localCollectReturn
}

func RegisterCollectReturn(i ICollectReturn) {
	localCollectReturn = i
}
