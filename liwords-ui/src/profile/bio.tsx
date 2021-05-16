import React from 'react';
import ReactMarkdown from 'react-markdown';
import { useMountedState } from '../utils/mounted';
import { useParams } from 'react-router-dom';
import { useLoginStateStoreContext } from '../store/store';

type BioProps = {
  bio: string;
  bioLoaded: boolean;
};

export const BioCard = React.memo((props: BioProps) => {
  const { useState } = useMountedState();
  const { loginState } = useLoginStateStoreContext();
  const { username: viewer } = loginState;
  const { username } = useParams();

  const [latestBio, setLatestBio] = useState('');

  React.useEffect(() => {
    setLatestBio(props.bio);
  }, [props.bio]);

  return viewer === username || latestBio !== '' ? (
    <div className="bio">
      <ReactMarkdown>
        {latestBio ? latestBio : "You haven't yet provided your bio."}
      </ReactMarkdown>
    </div>
  ) : null;
});
