import React from 'react';

type Props = {};

export const Preferences = React.memo((props: Props) => {
	return (
		<>
			<h3>Preferences</h3>
			<div>Display</div>
			<div>(Dark mode control)</div>
			<div>OMGWords settings</div>
			<div>Default tile order</div>
			selector
			<div>Score + clock placement</div>
			selector
			<div>Show letters remaining (with toggle)</div>
			<div>Languages</div>
			<div>English</div>
			<div>See English game requests (CSW)</div>
			<div>English</div>
			<div>See American English game requests (NWL)</div>
			<div>Polish</div>
			<div>See Polish game requests</div>
			<div>Norwegian</div>
			<div>See Norwegian game requests</div>
		</>
	);
});
