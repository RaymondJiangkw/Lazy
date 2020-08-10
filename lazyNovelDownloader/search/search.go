package search

import (
	"fmt"
	"io"

	"github.com/RaymondJiangkw/Lazy/utils"
)

const (
	searchItemNumbers = 10
	searchKey         = "目录"
	searchText        = "Searching for catalogs..."
)

func searchCatalogues(novelName string) ([]utils.TagA, error) {
	rets, err := utils.Search(&utils.SearchOption{Key: novelName + searchKey, Items: searchItemNumbers})
	if err == utils.Shortage && len(rets) == 0 {
		return nil, utils.Invalid
	}
	return rets, err
}

func Search(writer io.Writer, novelName string) ([]string, error) {
	var display utils.Display
	signal := make(chan struct{})
	finish := display.TemporaryText(writer, searchText, signal)
	potentialCatalogTags, err := searchCatalogues(novelName)
	signal <- struct{}{}
	<-finish
	if err != nil && len(potentialCatalogTags) == 0 {
		return nil, fmt.Errorf("Not Found Any Catalogues.")
	}
	rets := make([]string, len(potentialCatalogTags), len(potentialCatalogTags))
	for i, tag := range potentialCatalogTags {
		rets[i] = tag.Href
	}
	return rets, nil
}
