import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Unrace } from './unrace';
import {
  BriefProfile,
  BriefProfilesRequest,
} from '../gen/api/proto/user_service/user_service_pb';
import { useClient } from './hooks/connect';
import { ProfileService } from '../gen/api/proto/user_service/user_service_pb';

type CacheType = Map<string, { data: BriefProfile | null; expires: number }>;

const BriefProfilesContext = createContext<{
  cache: CacheType;
  request: (s: string) => void;
}>({
  cache: new Map(),
  request: () => {},
});

export const BriefProfiles = (props: {
  children?: JSX.Element | JSX.Element[];
}) => {
  const [profileId, setProfileId] = useState(0);
  const triggerRefresh = useCallback(
    () => setProfileId((n) => (n + 1) | 0),
    []
  );

  const cacheRef = useRef<CacheType>(new Map());
  const unrace = useRef(new Unrace());
  const timerToRequest = useRef<ReturnType<typeof setTimeout> | null>(null);
  const toRequest = useRef(new Set<string>());
  const profileClient = useClient(ProfileService);

  const performRequest = useCallback(() => {
    if (timerToRequest.current != null) {
      clearTimeout(timerToRequest.current);
      timerToRequest.current = null;
    }
    const toRequestHere = toRequest.current;
    unrace.current.run(async () => {
      // only run the latest batch
      if (toRequestHere !== toRequest.current) return;
      if (toRequestHere.size === 0) return;
      toRequest.current = new Set();
      const req = new BriefProfilesRequest();
      req.userIds = Array.from(toRequestHere);
      try {
        const respObj = await profileClient.getBriefProfiles(req);
        const respMap = respObj.response;
        // because of the await, toRequest.current may have accumulated more items
        const expires = performance.now() + 300000; // 5 minutes
        toRequestHere.forEach((k) => {
          cacheRef.current.set(k, { data: respMap[k] ?? null, expires });
          toRequest.current.delete(k); // this response also answers queries during await
        });
        triggerRefresh();
      } catch (e) {
        console.error('unable to access api', e);
      }
    });
  }, [triggerRefresh, profileClient]);
  const request = useCallback(
    (s: string) => {
      const cached = cacheRef.current.get(s);
      if (
        !(cached && cached.expires > performance.now()) &&
        !toRequest.current.has(s)
      ) {
        toRequest.current.add(s);
        if (timerToRequest.current == null)
          timerToRequest.current = setTimeout(performRequest, 0);
      }
    },
    [performRequest]
  );
  useEffect(() => {
    return () => {
      if (timerToRequest.current != null) clearTimeout(timerToRequest.current);
    };
  }, []);

  const cache = cacheRef.current;
  const value = useMemo(() => {
    void profileId; // we really want to return a new instance when this changes
    return { cache, request };
  }, [cache, request, profileId]);

  return (
    <BriefProfilesContext.Provider value={value}>
      {props.children}
    </BriefProfilesContext.Provider>
  );
};

export const useBriefProfile = (userID?: string) => {
  const briefProfiles = useContext(BriefProfilesContext);
  useEffect(() => {
    if (userID) briefProfiles.request(userID);
  }, [userID, briefProfiles]);

  return userID ? briefProfiles.cache.get(userID)?.data : undefined;
};
