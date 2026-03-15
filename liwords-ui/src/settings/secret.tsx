import React from "react";
import { Switch } from "antd";
import { useLocalStorageBool } from "../utils/use_local_storage";

export const Secret = React.memo(() => {
  const [telestrator, setTelestrator] = useLocalStorageBool(
    "enableScreenDrawing",
  );
  const [enableAllLexicons, setEnableAllLexicons] =
    useLocalStorageBool("enableAllLexicons");
  const [blindfold, setBlindfold] = useLocalStorageBool("enableBlindfoldMode");
  const [showEquityLoss, setShowEquityLoss] = useLocalStorageBool(
    "enableShowEquityLoss",
  );
  const [enableSilentSite, setEnableSilentSite] =
    useLocalStorageBool("enableSilentSite");
  const [hidePool, setHidePool] = useLocalStorageBool("hidePool");
  const [enableBicolorMode, setEnableBicolorMode] =
    useLocalStorageBool("enableBicolorMode");

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
              <div>Draw on the board while you're playing</div>
              <Switch
                checked={telestrator}
                onChange={setTelestrator}
                className="telestrator-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Blindfold</div>
            <div>
              <div>Enable text-to-speech keyboard commands</div>
              <Switch
                checked={blindfold}
                onChange={setBlindfold}
                className="blindfold-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Lexicons</div>
            <div>
              <div>Enable all lexicons</div>
              <Switch
                checked={enableAllLexicons}
                onChange={setEnableAllLexicons}
                className="dark-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Show equity loss</div>
            <div>
              <div>Show equity loss in analyzer</div>
              <Switch
                checked={showEquityLoss}
                onChange={setShowEquityLoss}
                className="show-equity-loss-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Enable silent site</div>
            <div>
              <div>Mute all sounds</div>
              <Switch
                checked={enableSilentSite}
                onChange={setEnableSilentSite}
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
                checked={hidePool}
                onChange={setHidePool}
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
                checked={enableBicolorMode}
                onChange={setEnableBicolorMode}
                className="bicolor-toggle"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
});
