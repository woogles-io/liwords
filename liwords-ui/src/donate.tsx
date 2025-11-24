import { loadStripe, StripeError } from "@stripe/stripe-js";
import { App, Button } from "antd";
import { useLoginStateStoreContext } from "./store/store";
import { LoginWithPatreonButton } from "./settings/integrations";
import ExternalLink from "./assets/external-link.svg?react";
import { useQuery } from "@connectrpc/connect-query";
import { getIntegrations } from "./gen/api/proto/user_service/user_service-IntegrationService_connectquery";
import { useCallback, useMemo } from "react";
import { getSubscriptionCriteria } from "./gen/api/proto/user_service/user_service-AuthorizationService_connectquery";

const PUBLISHABLE_KEY =
  "pk_live_51I7T0HH0ARGCjmpLmLvzN6JMTkUCaFr0xNhg7Mq2wcXTMhGI6R7ShMxnLmoaCynTO0cQ7BZtiSPfOjnA9LmO21dT00gBrlxiSa";

const prices = {
  5: "price_1Iaq0DH0ARGCjmpLkqP0dtl0",
  20: "price_1Iaq18H0ARGCjmpL1SV8SQff",
  50: "price_1Iaq1YH0ARGCjmpLfUUwAOdu",
  100: "price_1Iaq1uH0ARGCjmpL9lsPC3jJ",
  500: "price_1Ib7UJH0ARGCjmpLWP4pDmTs",
};

const DOMAIN = new URL("/", window.location.href).href;
const stripePromise = (async () => {
  try {
    return await loadStripe(PUBLISHABLE_KEY);
  } catch (e) {
    console.groupCollapsed("cannot load Stripe");
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

  const { data: userIntegrations } = useQuery(
    getIntegrations,
    {},
    { enabled: loginState.loggedIn },
  );

  const { data: subscriptionCriteria } = useQuery(
    getSubscriptionCriteria,
    {},
    { enabled: loginState.loggedIn },
  );

  const donateClick = async (money: number) => {
    const price = prices[money as keyof typeof prices];
    const mode = "payment";
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
        successUrl: DOMAIN + "donate_success?session_id={CHECKOUT_SESSION_ID}",
        cancelUrl: DOMAIN + "donate?session_id={CHECKOUT_SESSION_ID}",
        clientReferenceId: loginState.loggedIn
          ? loginState.userID + ":" + loginState.username
          : "anonymous-" + loginState.userID,
        submitType: "donate",
      })
      .then(handleResult);
  };

  const hasPatreonIntegration = useMemo(
    () =>
      userIntegrations?.integrations.some(
        (i) => i.integrationName === "patreon",
      ),
    [userIntegrations],
  );

  const patreonButton = useCallback(() => {
    if (!loginState.loggedIn) {
      return (
        <p>
          Please log in to Woogles to connect your Patreon account after
          subscribing.
        </p>
      );
    }
    if (hasPatreonIntegration) {
      if (subscriptionCriteria?.tierName) {
        return (
          <p>
            You are subscribed at the {subscriptionCriteria.tierName} level.
            Thank you!
          </p>
        );
      }
      return (
        <p style={{ marginTop: 10 }}>
          <Button
            style={{ width: 256 }}
            onClick={() => {
              window.location.href = "https://patreon.com/woogles_io";
            }}
          >
            <ExternalLink className="pt-callout-link" />
            Subscribe monthly on Patreon
          </Button>
        </p>
      );
    } else {
      return (
        <p style={{ marginTop: 10 }}>
          <LoginWithPatreonButton
            label="Link your Patreon account"
            style={{ width: 300 }}
          />
        </p>
      );
    }
  }, [loginState.loggedIn, hasPatreonIntegration, subscriptionCriteria]);

  return (
    <>
      <p>
        We’re an entirely volunteer-run 501(c)(3) non-profit. If you’re enjoying
        the site, please consider contributing via a one-time donation or
        monthly subscription.
      </p>
      <p className="bolder" style={{ marginTop: 24 }}>
        One-time donation
      </p>
      <p>
        Tax-deductible, one-time donations do not come with any site benefits
      </p>
      <div className="donation-buttons">
        <Button onClick={() => donateClick(5)}>Contribute $5</Button>
        <Button onClick={() => donateClick(20)}>Contribute $20</Button>
        <Button onClick={() => donateClick(50)}>Contribute $50</Button>
        <Button onClick={() => donateClick(100)}>Contribute $100</Button>
        <Button onClick={() => donateClick(500)}>Contribute $500</Button>
      </div>
      <p className="bolder">Subscribe monthly</p>
      <p>
        When you join the Woogles Patreon, you can get access to BestBot, cool
        badges, and even Woogles swag!
      </p>
      {patreonButton()}
    </>
  );
};
