import React from 'react';

import { Button } from 'antd';

type Props = {};

export const Support = React.memo((props: Props) => {
	return (
		<>
			<h3>Support Woogles.io</h3>
			<div>Need Woogles doggie image</div>
			<div>Help us keep Woogles.io going!</div>
			<div>
				We’re an entirely volunteer-run 503(c) NFP. If you’re enjoying the site,
				please feel free to contribute a few dollars to us!
			</div>
			<Button>Contribute</Button>
		</>
	);
});
