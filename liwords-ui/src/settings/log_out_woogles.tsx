import React from 'react';

import { Button } from 'antd';

type Props = {};

export const LogOut = React.memo((props: Props) => {
	return (
		<>
			<h3>Log out of Woogles.io</h3>
			<div>Avatar and screen name go here...</div>
			<div>
				You’ll have to log back in to your account to play games or see tiles
				while watching tournament games on Woogles.io.
			</div>
			<Button>Log out</Button>
		</>
	);
});
