import React from 'react';

import { loadStripe } from '@stripe/stripe-js';
import { TopBar } from './topbar/topbar';
import { Button, Col, message, Row } from 'antd';
import { useLoginStateStoreContext } from './store/store';

const PUBLISHABLE_KEY = window.RUNTIME_CONFIGURATION.stripePublishableKey;

// Use production products after testing
const prices = window.RUNTIME_CONFIGURATION.stripePrices;

const DOMAIN = window.location.href.replace(/[^/]*$/, '');
const stripePromise = loadStripe(PUBLISHABLE_KEY);

export const Donate = () => {
  const { loginState } = useLoginStateStoreContext();

  const handleResult = (result: any) => {
    if (result.error) {
      message.error({
        content: result.error.message,
        duration: 5,
      });
    }
  };

  const donateClick = async (money: number) => {
    const price = prices[money];
    const mode = 'payment';
    const items = [
      {
        price,
        quantity: 1,
      },
    ];
    const stripe = await stripePromise;
    if (!stripe) {
      return;
    }
    await stripe
      .redirectToCheckout({
        mode,
        lineItems: items,
        successUrl: DOMAIN + 'donate_success?session_id={CHECKOUT_SESSION_ID}',
        cancelUrl: DOMAIN + 'donate?session_id={CHECKOUT_SESSION_ID}',
        clientReferenceId: loginState.loggedIn
          ? loginState.userID + ':' + loginState.username
          : 'anonymous' + '-' + loginState.userID,
        submitType: 'donate',
      })
      .then(handleResult);
  };

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <div className="donations">
        <p>
          We really appreciate your donations. You can select one-time
          donations, or donate via our Patreon account
        </p>
        <p>
          <Button onClick={() => donateClick(5)}>Donate $5 one-time</Button>
          <Button onClick={() => donateClick(10)}>Donate $10 one-time</Button>
          <Button onClick={() => donateClick(20)}>Donate $20 one-time</Button>
          <Button onClick={() => donateClick(50)}>Donate $50 one-time</Button>
          <Button onClick={() => donateClick(100)}>Donate $100 one-time</Button>
        </p>
      </div>
      <p></p>
      <div>
        <p>
          You can also donate using our Patreon account, for recurring payments:{' '}
          <a href="https://www.patreon.com/woogles_io">
            https://www.patreon.com/woogles_io
          </a>
        </p>
      </div>
    </>
  );
};
