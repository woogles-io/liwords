import React from 'react';
import { Link, useHistory } from 'react-router-dom';
import { useMountedState } from '../utils/mounted';
import axios from 'axios';
import { TopBar } from '../topbar/topbar';
import { Input, Form, Button, Alert, Checkbox, Select } from 'antd';
import { Rule } from 'antd/lib/form';
import { toAPIUrl } from '../api/api';
import './accountForms.scss';
import woogles from '../assets/woogles.png';
import { countryArray } from '../settings/country_map';

const usernameValidator = async (rule: Rule, value: string) => {
  if (!value) {
    throw new Error('Please input your username');
  }
  if (value.length < 3) {
    throw new Error('Min username length is 3');
  }
  if (value.length > 20) {
    throw new Error('Max username length is 20');
  }
  if (!/^[0-9a-zA-Z\-_.]+$/.test(value)) {
    throw new Error('Valid characters are A-Z a-z 0-9 . _ -');
  }
  if (!/^[0-9a-zA-Z]/.test(value)) {
    throw new Error('Valid starting characters are A-Z a-z 0-9');
  }
  if (!/[0-9a-zA-Z]$/.test(value)) {
    throw new Error('Valid ending characters are A-Z a-z 0-9');
  }
};

const birthDateValidator = async (rule: Rule, value: string) => {
  let valid = false;
  if (value) {
    const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
    if (match) {
      const [, sy, sm, sd] = match;
      const iy = +sy;
      const im = +sm - 1; // zero-based month
      const id = +sd;
      // TODO: check for allowed range?
      const d = new Date(Date.UTC(iy, im, id));
      if (
        d.getUTCFullYear() === iy &&
        d.getUTCMonth() === im &&
        d.getUTCDate() === id
      ) {
        valid = true;
      }
    }
  }
  if (!valid) {
    throw new Error('Please input your birth date in YYYY-MM-DD');
  }
};

export const Register = () => {
  const { useState } = useMountedState();

  const [err, setErr] = useState('');
  const [loggedIn, setLoggedIn] = useState(false);

  const onFinish = (values: { [key: string]: string }) => {
    axios
      .post(toAPIUrl('user_service.RegistrationService', 'Register'), {
        username: values.username,
        password: values.password,
        email: values.email,
        registrationCode: values.registrationCode,
        birthDate: values.birthDate,
        firstName: values.firstName,
        lastName: values.lastName,
        countryCode: values.countryCode,
      })
      .then(() => {
        // Try logging in after registering.
        axios
          .post(
            toAPIUrl('user_service.AuthenticationService', 'Login'),
            {
              username: values.username,
              password: values.password,
            },
            { withCredentials: true }
          )
          .then(() => {
            // Automatically will set cookie
            setLoggedIn(true);
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

  const history = useHistory();
  React.useEffect(() => {
    if (loggedIn) {
      history.replace('/');
    }
  }, [history, loggedIn]);

  // XXX: This is copied from settings/personal_info.tsx (with added placeholder).
  // It has the same issues (such as the emoji not displaying on Windows).
  const countrySelector = (
    <Select size="large" bordered={false} placeholder="Country">
      {countryArray.map((country) => {
        return (
          <Select.Option key={country.code} value={country.code.toLowerCase()}>
            {country.emoji} {country.name}
          </Select.Option>
        );
      })}
    </Select>
  );

  return (
    <>
      <TopBar />
      <div className="account">
        <img src={woogles} className="woogles" alt="Woogles" />
        <div className="account-form register">
          <h3>Welcome to Woogles!</h3>
          <p>
            Welcome to Woogles, the online home for word games! If you want to
            be the champion of crossword game, or maybe just want to find a
            friendly match you’re in the right place. By the way, signing up
            means you agree to our{' '}
            <Link target="_blank" to="/about">
              Privacy Policy
            </Link>
            .
          </p>
          <p>- Woogles and team</p>
          <Form layout="inline" name="register" onFinish={onFinish}>
            <p className="group-title">Required information</p>
            <Form.Item
              name="email"
              hasFeedback
              rules={[
                {
                  required: true,
                  message: "We need your email. We won't spam you",
                },
                {
                  type: 'email',
                  message: 'This is not a valid email',
                },
              ]}
            >
              <Input placeholder="Email" />
            </Form.Item>
            <Form.Item
              name="username"
              hasFeedback
              rules={[
                {
                  validator: usernameValidator,
                },
              ]}
            >
              <Input placeholder="Username" maxLength={20} />
            </Form.Item>

            <Form.Item
              name="password"
              className="password"
              hasFeedback
              rules={[
                {
                  required: true,
                  message: 'Please input your password',
                },
                {
                  min: 8,
                  message: 'Password should be at least 8 characters',
                },
              ]}
            >
              <Input.Password
                placeholder="Password"
                autoComplete="new-password"
              />
            </Form.Item>

            <p className="persistent-explanation">
              Password must include at least one lowercase letter, upper case
              letter, number, and symbol.
            </p>

            {/* This is probably obsolete but in case we have to pause registration
            <Form.Item
              name="registrationCode"
              hasFeedback
              rules={[
                {
                  required: true,
                  message: 'You need a registration code',
                },
              ]}
            >
              <Input placeholder="Secret code" />
            </Form.Item>
            */}

            <Form.Item
              name="birthDate"
              hasFeedback
              rules={[
                {
                  validator: birthDateValidator,
                },
              ]}
            >
              <Input placeholder="Birth date (YYYY-MM-DD)" />
            </Form.Item>

            <p className="persistent-explanation">
              Birthday info will never be displayed on Woogles.
            </p>

            <p className="group-title">Optional info</p>

            <Form.Item name="firstName">
              <Input placeholder="First name" />
            </Form.Item>

            <Form.Item name="lastName">
              <Input placeholder="Last name" />
            </Form.Item>

            <span className="country-code">
              <Form.Item name="countryCode">{countrySelector}</Form.Item>
            </span>

            <span className="full-width">
              <Form.Item
                rules={[
                  {
                    required: true,
                    message: 'You must agree to this condition',
                    transform: (value) => value || undefined,
                    type: 'boolean',
                  },
                ]}
                valuePropName="checked"
                initialValue={false}
                name="nocheating"
              >
                <Checkbox>
                  <p className="no-cheat">
                    I agree to the{' '}
                    <Link target="_blank" to="/about">
                      Woogles Terms of Service
                    </Link>
                    . Notably, I promise not to use word finders or game
                    analyzers in rated games.
                  </p>
                </Checkbox>
              </Form.Item>
            </span>

            <Form.Item>
              <Button type="primary" htmlType="submit">
                Continue
              </Button>
            </Form.Item>
          </Form>
          {err !== '' ? <Alert message={err} type="error" /> : null}

          <p>
            Already have a Woogles account? No worries!{' '}
            <Link to="/">Log in here</Link>
          </p>
        </div>
      </div>
    </>
  );
};
