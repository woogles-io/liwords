import React from 'react';

import { Button, Input } from 'antd';

type Props = {};

export const ChangePassword = React.memo((props: Props) => {
	return (
		<>
			<h3>Change password</h3>
			Old password
			<Input />
			New password
			<Input />
			<Button>Submit</Button>
		</>
	);
});
