import { loadStripe, StripeError } from '@stripe/stripe-js';
import { App, Button } from 'antd';
import { useLoginStateStoreContext } from './store/store';

const PUBLISHABLE_KEY =
  'pk_live_51I7T0HH0ARGCjmpLmLvzN6JMTkUCaFr0xNhg7Mq2wcXTMhGI6R7ShMxnLmoaCynTO0cQ7BZtiSPfOjnA9LmO21dT00gBrlxiSa';

const prices = {
  5: 'price_1Iaq0DH0ARGCjmpLkqP0dtl0',
  20: 'price_1Iaq18H0ARGCjmpL1SV8SQff',
  50: 'price_1Iaq1YH0ARGCjmpLfUUwAOdu',
  100: 'price_1Iaq1uH0ARGCjmpL9lsPC3jJ',
  500: 'price_1Ib7UJH0ARGCjmpLWP4pDmTs',
};

const DOMAIN = new URL('/', window.location.href).href;
const stripePromise = (async () => {
  try {
    return await loadStripe(PUBLISHABLE_KEY);
  } catch (e) {
    console.groupCollapsed('cannot load Stripe');
    console.error(e);
    console.groupEnd();
    return null;
  }
})();

type StripeResult = {
  error?: StripeError;
};

export const Donate = () => {
  const { loginState } = useLoginStateStoreContext();
  const { message } = App.useApp();
  const handleResult = (result: StripeResult) => {
    if (result.error) {
      message.error({
        content: result.error.message,
        duration: 5,
      });
    }
  };

  const donateClick = async (money: number) => {
    const price = prices[money as keyof typeof prices];
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
          : 'anonymous-' + loginState.userID,
        submitType: 'donate',
      })
      .then(handleResult);
  };

  return (
    <>
      <div className="title">Help us keep Woogles.io going!</div>
      <p>
        We’re an entirely volunteer-run 501(c)(3) non-profit. If you’re enjoying
        the site, please feel free to contribute a few dollars to us!
      </p>
      <div className="donation-buttons">
        <Button onClick={() => donateClick(5)}>Contribute $5</Button>
        <Button onClick={() => donateClick(20)}>Contribute $20</Button>
        <Button onClick={() => donateClick(50)}>Contribute $50</Button>
        <Button onClick={() => donateClick(100)}>Contribute $100</Button>
        <Button onClick={() => donateClick(500)}>Contribute $500</Button>
      </div>
      <p>
        <span className="bolder">Want to make a monthly donation?</span> Check
        out the
        <a href="https://www.patreon.com/woogles_io"> Woogles Patreon.</a>
      </p>
    </>
  );
};
