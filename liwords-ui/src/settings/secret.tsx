import React, { useCallback, useState } from "react";
import { Switch } from "antd";

export const Secret = React.memo(() => {
  const [telestrator, setTelestrator] = useState(
    localStorage?.getItem("enableScreenDrawing") === "true",
  );
  const toggleTelestrator = useCallback(() => {
    const useTelestrator =
      localStorage?.getItem("enableScreenDrawing") !== "true";
    localStorage.setItem(
      "enableScreenDrawing",
      useTelestrator ? "true" : "false",
    );
    setTelestrator((x) => !x);
  }, []);

  const [enableAllLexicons, setEnableAllLexicons] = useState(
    localStorage?.getItem("enableAllLexicons") === "true",
  );
  const toggleEnableAllLexicons = useCallback(() => {
    const wantEnableAllLexicons =
      localStorage?.getItem("enableAllLexicons") !== "true";
    localStorage.setItem(
      "enableAllLexicons",
      wantEnableAllLexicons ? "true" : "false",
    );
    setEnableAllLexicons((x) => !x);
  }, []);

  const [blindfold, setBlindfold] = useState(
    localStorage?.getItem("enableBlindfoldMode") === "true",
  );
  const toggleBlindfold = useCallback(() => {
    const useBlindfold =
      localStorage?.getItem("enableBlindfoldMode") !== "true";
    localStorage.setItem(
      "enableBlindfoldMode",
      useBlindfold ? "true" : "false",
    );
    setBlindfold((x) => !x);
  }, []);

  const [showEquityLoss, setShowEquityLoss] = useState(
    localStorage?.getItem("enableShowEquityLoss") === "true",
  );
  const toggleShowEquityLoss = useCallback(() => {
    const useShowEquityLoss =
      localStorage?.getItem("enableShowEquityLoss") !== "true";
    localStorage.setItem(
      "enableShowEquityLoss",
      useShowEquityLoss ? "true" : "false",
    );
    setShowEquityLoss((x) => !x);
  }, []);

  const [enableSilentSite, setEnableSilentSite] = useState(
    localStorage?.getItem("enableSilentSite") === "true",
  );
  const toggleEnableSilentSite = useCallback(() => {
    const wantEnableSilentSite =
      localStorage?.getItem("enableSilentSite") !== "true";
    localStorage.setItem(
      "enableSilentSite",
      wantEnableSilentSite ? "true" : "false",
    );
    setEnableSilentSite((x) => !x);
  }, []);

  const [hidePool, setHidePool] = useState(
    localStorage?.getItem("hidePool") === "true",
  );
  const toggleHidePool = useCallback(() => {
    const wantHidePool = localStorage?.getItem("hidePool") !== "true";
    localStorage.setItem("hidePool", wantHidePool ? "true" : "false");
    setHidePool((x) => !x);
  }, []);

  const [enableBicolorMode, setEnableBicolorMode] = useState(
    localStorage?.getItem("enableBicolorMode") === "true",
  );
  const toggleEnableBicolorMode = useCallback(() => {
    const wantEnableBicolorMode =
      localStorage?.getItem("enableBicolorMode") !== "true";
    localStorage.setItem(
      "enableBicolorMode",
      wantEnableBicolorMode ? "true" : "false",
    );
    setEnableBicolorMode((x) => !x);
  }, []);

  return (
    <div className="preferences secret">
      <h3>Secret features</h3>
      <div className="secret-warning">
        Please use these secret, experimental features at your own discretion.
        They may be limited in functionality and/or impact your Woogles user
        experience.{" "}
        <a
          href="https://github.com/woogles-io/liwords/wiki/Secret-features"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn more.
        </a>
      </div>
      <div className="toggles-section">
        <div>
          <div className="toggle-section">
            <div className="title">Telestrator</div>
            <div>
              <div>Draw on the board while you’re playing</div>
              <Switch
                defaultChecked={telestrator}
                onChange={toggleTelestrator}
                className="telestrator-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Blindfold</div>
            <div>
              <div>Enable text-to-speech keyboard commands</div>
              <Switch
                defaultChecked={blindfold}
                onChange={toggleBlindfold}
                className="blindfold-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Lexicons</div>
            <div>
              <div>Enable all lexicons</div>
              <Switch
                defaultChecked={enableAllLexicons}
                onChange={toggleEnableAllLexicons}
                className="dark-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Show equity loss</div>
            <div>
              <div>Show equity loss in analyzer</div>
              <Switch
                defaultChecked={showEquityLoss}
                onChange={toggleShowEquityLoss}
                className="show-equity-loss-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Enable silent site</div>
            <div>
              <div>Mute all sounds</div>
              <Switch
                defaultChecked={enableSilentSite}
                onChange={toggleEnableSilentSite}
                className="sounds-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Practice manual tracking and scoring</div>
            <div>
              <div>
                Disable automatic tracking of tiles and scoring of tentative
                moves for you only
              </div>
              <Switch
                defaultChecked={hidePool}
                onChange={toggleHidePool}
                className="pool-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Infuse Second Color</div>
            <div>
              <div>
                Highlight one player's tiles instead of the last move. Requires
                Refresher Orb.
              </div>
              <Switch
                defaultChecked={enableBicolorMode}
                onChange={toggleEnableBicolorMode}
                className="bicolor-toggle"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
});
