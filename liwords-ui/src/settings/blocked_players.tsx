import React from 'react';

type Props = {};

export const BlockedPlayers = React.memo((props: Props) => {
	return (
		<>
			<h3>Blocked players list</h3>
			<div>
				This is the list of players you’ve blocked on Woogles.io. Blocked
				players can’t see that you’re online, and you can’t see that they’re
				online.
			</div>
			<div>
				You will, however, still be able to see each others profiles and past
				games
			</div>
			<div>x Bob</div>
			<div>x Joe</div>
		</>
	);
});
