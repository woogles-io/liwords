import { useCallback } from "react";
import { useSearchParams, useNavigate } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { getCollection } from "../gen/api/proto/collections_service/collections_service-CollectionsService_connectquery";
import { Collection } from "../gen/api/proto/collections_service/collections_service_pb";

export interface CollectionContext {
  collectionUuid: string;
  currentChapter: number;
  totalChapters: number;
  collection: Collection | null;
  loading: boolean;
  error: string | null;

  // Navigation functions
  goToChapter: (chapter: number) => void;
  goToPreviousChapter: () => void;
  goToNextChapter: () => void;
  goBackToCollection: () => void;

  // State flags
  hasPrevious: boolean;
  hasNext: boolean;
}

export const useCollectionContext = (): CollectionContext | null => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  // Parse collection context from URL
  const collectionUuid = searchParams.get("collection");
  const chapterParam = searchParams.get("chapter");
  const totalParam = searchParams.get("total");

  const currentChapter = parseInt(chapterParam || "1", 10);
  const totalChapters = parseInt(totalParam || "1", 10);

  // Check if we have collection context
  const hasCollectionContext = !!(collectionUuid && chapterParam && totalParam);

  // Fetch collection data using useQuery with built-in caching
  const {
    data: response,
    isLoading: loading,
    error: queryError,
  } = useQuery(
    getCollection,
    { collectionUuid: collectionUuid || "" },
    { enabled: hasCollectionContext && !!collectionUuid },
  );

  const collection = response?.collection || null;
  const error = queryError ? "Failed to load collection" : null;

  // Navigation functions
  const goToChapter = useCallback(
    (chapter: number) => {
      if (!collection || chapter < 1 || chapter > collection.games.length)
        return;

      const targetGame = collection.games[chapter - 1];
      const baseUrl = targetGame.isAnnotated
        ? `/anno/${targetGame.gameId}`
        : `/game/${targetGame.gameId}`;

      // Preserve existing search parameters (like 'turn') when navigating
      const params = new URLSearchParams(searchParams);
      params.set("collection", collectionUuid!);
      params.set("chapter", chapter.toString());
      params.set("total", collection.games.length.toString());

      navigate(`${baseUrl}?${params.toString()}`);
    },
    [collection, collectionUuid, navigate, searchParams],
  );

  const goToPreviousChapter = useCallback(() => {
    if (currentChapter > 1) {
      goToChapter(currentChapter - 1);
    }
  }, [currentChapter, goToChapter]);

  const goToNextChapter = useCallback(() => {
    if (currentChapter < totalChapters) {
      goToChapter(currentChapter + 1);
    }
  }, [currentChapter, totalChapters, goToChapter]);

  const goBackToCollection = useCallback(() => {
    if (!collectionUuid) return;
    navigate(`/collections/${collectionUuid}/chapter/${currentChapter}`);
  }, [navigate, collectionUuid, currentChapter]);

  // Return null if no collection context
  if (!hasCollectionContext) {
    return null;
  }

  const hasPrevious = currentChapter > 1;
  const hasNext = currentChapter < totalChapters;

  return {
    collectionUuid: collectionUuid!,
    currentChapter,
    totalChapters,
    collection,
    loading,
    error,
    goToChapter,
    goToPreviousChapter,
    goToNextChapter,
    goBackToCollection,
    hasPrevious,
    hasNext,
  };
};
