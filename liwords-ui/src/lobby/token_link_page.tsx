import React, { ReactNode, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { Spin, Result } from "antd";
import { TopBar } from "../navigation/topbar";
import { connectErrorMessage } from "../utils/hooks/connect";
import "./accountForms.scss";

type Props<T> = {
  // Sends the token from the link to the server. It is called once per
  // token, even when React runs effects twice in development, since some
  // tokens work only once.
  confirm: (token: string) => Promise<T>;
  loadingText: string;
  successTitle: string;
  successSubTitle: (result: T) => ReactNode;
  successExtra: ReactNode[];
  errorTitle: string;
  errorSubTitle: (message: string) => ReactNode;
  errorExtra: (message: string) => ReactNode[];
  // Shown under the error, if anything.
  errorDetail?: (message: string) => ReactNode;
  // Rendered after the page, for things like modals.
  children?: ReactNode;
};

// TokenLinkPage is where a link with a ?token= from an email lands: it
// confirms the token and says how that went.
export const TokenLinkPage = <T,>(props: Props<T>) => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token");
  const [state, setState] = useState<
    | { status: "loading" }
    | { status: "success"; result: T }
    | { status: "error"; message: string }
  >(token ? { status: "loading" } : missingToken);

  // The latest confirm, so the effect below depends only on the token.
  const confirmRef = useRef(props.confirm);
  confirmRef.current = props.confirm;
  const startedFor = useRef<string | null>(null);

  useEffect(() => {
    if (!token) {
      setState(missingToken);
      return;
    }
    if (startedFor.current === token) {
      return;
    }
    startedFor.current = token;
    setState({ status: "loading" });
    confirmRef.current(token).then(
      (result) => setState({ status: "success", result }),
      (e) => setState({ status: "error", message: connectErrorMessage(e) }),
    );
  }, [token]);

  return (
    <>
      <TopBar />
      <div className="account-form">
        <div className="form-container">
          {state.status === "loading" && (
            <div style={{ textAlign: "center", padding: "60px 0" }}>
              <Spin size="large" />
              <p style={{ marginTop: "20px", fontSize: "16px" }}>
                {props.loadingText}
              </p>
            </div>
          )}

          {state.status === "success" && (
            <Result
              status="success"
              title={props.successTitle}
              subTitle={props.successSubTitle(state.result)}
              extra={props.successExtra}
            />
          )}

          {state.status === "error" && (
            <Result
              status="error"
              title={props.errorTitle}
              subTitle={props.errorSubTitle(state.message)}
              extra={props.errorExtra(state.message)}
            >
              {props.errorDetail?.(state.message)}
            </Result>
          )}
        </div>
      </div>
      {props.children}
    </>
  );
};

const missingToken = {
  status: "error" as const,
  message: "This link has no token in it.",
};
