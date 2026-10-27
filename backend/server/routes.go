package server

import (
	"net/http"
	"path"
	"strings"

	"abridged-web/backend/catalog"
)

type routeKind int

const (
	routeKindStatic routeKind = iota
	routeKindHome
	routeKindBrowse
	routeKindCreators
	routeKindCreator
	routeKindEntry
	routeKindEpisode
	routeKindNotFound
)

type resolvedRoute struct {
	kind        routeKind
	status      int
	entryType   catalog.EntryType
	creatorSlug string
	entry       *catalog.Entry
	episode     *catalog.Episode
}

// resolveRoute mirrors frontend/src/lib/router.ts. It is shared by the status
// resolver and the metadata injector so both agree on what a URL represents.
func resolveRoute(library *catalog.Library, requestPath string) resolvedRoute {
	cleanPath := path.Clean(requestPath)
	if cleanPath == "/" || cleanPath == "." || cleanPath == "/index.html" {
		return resolvedRoute{kind: routeKindHome, status: http.StatusOK}
	}
	if path.Ext(cleanPath) != "" {
		return resolvedRoute{kind: routeKindStatic, status: http.StatusOK}
	}

	segments := strings.Split(strings.Trim(cleanPath, "/"), "/")
	if len(segments) == 0 {
		return resolvedRoute{kind: routeKindHome, status: http.StatusOK}
	}

	switch len(segments) {
	case 1:
		switch segments[0] {
		case "series":
			return resolvedRoute{kind: routeKindBrowse, status: http.StatusOK, entryType: catalog.EntryTypeSeries}
		case "shorts":
			return resolvedRoute{kind: routeKindBrowse, status: http.StatusOK, entryType: catalog.EntryTypeShort}
		case "shots":
			return resolvedRoute{kind: routeKindBrowse, status: http.StatusOK, entryType: catalog.EntryTypeShot}
		case "songs":
			return resolvedRoute{kind: routeKindBrowse, status: http.StatusOK, entryType: catalog.EntryTypeSong}
		case "creators":
			return resolvedRoute{kind: routeKindCreators, status: http.StatusOK}
		}
		if library.HasCreator(segments[0]) {
			return resolvedRoute{kind: routeKindCreator, status: http.StatusOK, creatorSlug: segments[0]}
		}

	case 2:
		if entry, ok := library.EntryForWatchPath(segments[0], segments[1]); ok {
			return resolvedRoute{kind: routeKindEntry, status: http.StatusOK, entry: &entry}
		}

	case 3:
		if entry, ok := library.EntryForWatchPath(segments[0], segments[1]); ok {
			if episode, ok := library.EpisodeForWatchPath(entry, segments[2]); ok {
				return resolvedRoute{kind: routeKindEpisode, status: http.StatusOK, entry: &entry, episode: &episode}
			}
		}
	}

	return resolvedRoute{kind: routeKindNotFound, status: http.StatusNotFound}
}

func resolveRouteStatus(library *catalog.Library, requestPath string) int {
	return resolveRoute(library, requestPath).status
}
