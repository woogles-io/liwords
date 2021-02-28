import React from 'react';
import { useMountedState } from '../utils/mounted';
import { Button, Input, Form, Alert, notification } from 'antd';
import axios from 'axios';
import { toAPIUrl } from '../api/api';

const layout = {
	labelCol: {
		span: 8,
	},
	wrapperCol: {
		span: 12,
	},
};

const tailLayout = {
	wrapperCol: {
		offset: 8,
		span: 12,
	},
};

type Props = {};

export const ChangePassword = React.memo((props: Props) => {
	const { useState } = useMountedState();
	const [err, setErr] = useState('');
	const [form] = Form.useForm();

	const onFinish = (values: { [key: string]: string }) => {
		axios
			.post(
				toAPIUrl('user_service.AuthenticationService', 'ChangePassword'),
				{
					oldPassword: values.oldPassword,
					newPassword: values.newPassword,
				},
				{
					withCredentials: true,
				}
			)
			.then(() => {
				notification.info({
					message: 'Success',
					description: 'Your password was changed.',
				});
				form.resetFields();
				setErr('');
			})
			.catch((e) => {
				if (e.response) {
					// From Twirp
					setErr(e.response.data.msg);
				} else {
					setErr('unknown error, see console');
					console.log(e);
				}
			});
	};

	return (
		<div className="change-password">
			<h3>Change password</h3>
			<Form
				form={form}
				{...layout}
				name="changepassword"
				onFinish={onFinish}
				style={{ marginTop: 20 }}
			>
				<Form.Item
					label="Old Password"
					name="oldPassword"
					rules={[
						{
							required: true,
							message: 'Please input your old password!',
						},
					]}
				>
					<Input.Password />
				</Form.Item>

				<Form.Item
					label="New Password"
					name="newPassword"
					rules={[
						{
							required: true,
							message: 'Please input your new password!',
						},
					]}
				>
					<Input.Password />
				</Form.Item>

				<Form.Item {...tailLayout}>
					<Button type="primary" htmlType="submit">
						Save
					</Button>
				</Form.Item>
			</Form>
			{err !== '' ? <Alert message={err} type="error" /> : null}
		</div>
	);
});
