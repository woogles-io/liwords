import React, { useEffect } from 'react';
import { TopBar } from '../topbar/topbar';
import './staticContent.scss';
import woogles from '../assets/woogles.png';
import moment from 'moment';

// Always change this when the terms change.
const TERMS_VERSION_DATE = '2021-07-06';

export const setTermsNotify = () => {
  localStorage?.setItem('termsDate', Date.parse(TERMS_VERSION_DATE).toString());
};

export const getTermsNotify = () => {
  const releaseDate = Date.parse(TERMS_VERSION_DATE);
  const termsDate = parseInt(localStorage?.getItem('termsDate') || '0', 10);
  return {
    notified: releaseDate === termsDate,
    changed: moment(TERMS_VERSION_DATE).format('MMMM Do, YYYY'),
  };
};

type Props = {};

export const TermsOfService = (props: Props) => {
  useEffect(() => {
    setTermsNotify();
  }, []);
  return (
    <>
      <TopBar />
      <div className="static-content">
        <img src={woogles} className="woogles" alt="Woogles" />
        <main>
          <h3>Woogles.io Terms of Service</h3>
          <h4>Welcome to Woogles.io!</h4>
          <p>
            This page explains our terms of use. When you use Woogles.io, you’re
            agreeing to all of the rules on this page.
          </p>
          <p>
            By using this website (the “Site”) and services (together with the
            Site, the “Services”) offered by Woogles.io (together with our
            subsidiaries, affiliates, representatives, and directors –
            collectively “Woogles”, “we”, or “us”) you’re agreeing to these
            legally binding rules (the “Terms”). You’re also agreeing to our
            Privacy Policy and Cookie Policy, and agreeing to follow any other
            rules on the Site, like our Fair Play rules (included below).
          </p>
          <p>
            We may change these terms from time to time. If we do, we’ll let you
            know about any significant or material changes by notifying you on
            the Site. New versions of the terms will never apply retroactively –
            we’ll tell you the exact date they go into effect. If you continue
            using Woogles after a change, that means you accept the new terms.
          </p>
          <h4>Introduction</h4>
          <p>
            Woogles is a platform intended to provide a fun, welcoming place for
            word gamers of all skill levels to compete and improve at their
            favorite games. We actively solicit feedback for features that our
            users would like to see. Please feel free to contact us with
            suggestions via our
            <a
              href="https://tinyurl.com/y4dkb2g6"
              target="_blank"
              rel="noopener noreferrer"
            >
              {' '}
              Google form
            </a>
            ,
            <a
              href="https://discord.gg/GqkUqA7ENm"
              target="_blank"
              rel="noopener noreferrer"
            >
              {' '}
              Discord,
            </a>{' '}
            or e-mail us at woogles at woogles.io.
          </p>
          <p>
            Woogles Inc is a recognized US non-profit in the 501(c)(3) sense
            incorporated in the State of New Jersey. Woogles features an open
            source codebase, is operated by volunteers, and is funded entirely
            by donations. The Site’s costs and expenses will be made publicly
            available.
          </p>
          <h4>Creating an Account</h4>
          <p>
            You can use many of Woogles’ Services without registering your
            account, but others, including playing games, will require you to
            register. To register, you will choose a username and a password and
            validate them with an email address. If you submit inaccurate or
            incomplete information, you might not be able to access your
            account.
          </p>
          <p>
            Don’t impersonate anyone else or choose offensive names. Woogles
            reserves the right to close your account if you fail to abide by
            either of these guidelines.
          </p>
          <p>
            “Users” or a “User” refer to both registered and unregistered users,
            unless it relates to features, information or data an unregistered
            User does not or cannot have access to.
          </p>
          <h4>User Rights</h4>
          <p>
            Both registered and unregistered users have rights to protect their
            property, privacy, and identity unless they have consented or agree
            to making their identity public. Woogles believes that users should
            have their public information, identity, and data strongly
            protected. We reserve the right in future to clarify these
            protections via a privacy policy, in supplement to these Terms and
            conditions, to specify which data Woogles will take from a User’s
            device and why it does so.
          </p>
          <p>
            By agreeing to these Terms, registered Users can have access to all
            of Woogles’ features and are free to use them for their own
            personal, educational, charitable, or developmental purposes.
            Although Woogles is open-source software licensed under the GNU AGPL
            license, Woogles still retains the right to make decisions
            concerning appropriate use of Services.
          </p>
          <p>
            Woogles is not responsible for the content that Users post. Our
            community monitors have the power to delete messages and ban users
            in a manner that is consistent with these Terms.
          </p>
          <p>
            In all circumstances, we may retain certain information as required
            by law or necessary for our legitimate operating purposes. We will
            never share your data unless in response to proper legal process or
            a proper request from an authority. All of the provisions within
            this agreement survive the termination of an account.
          </p>
          <h4>Fair Play and Community Guidelines</h4>
          <p>
            All users (registered and unregistered) agree to behave with good
            conduct whilst using Woogles’ Services. This will always be
            determined at Woogles’ discretion. Users who don’t behave with good
            conduct may have their account suspended or closed without warning.
            In all circumstances we reserve the right to suspend your account,
            or close it for any reason, without warning and without having to
            provide evidence that Fair Play and Community Guidelines have been
            breached.
          </p>
          <p>
            A user whose account has been suspended or closed may apply for
            reinstatement by e-mailing conduct at woogles.io. We reserve the
            right to reinstate previously suspended or banned accounts without
            providing a reason, and also reserve the right to deny requests for
            reinstatement without providing an explanation for the denial.
          </p>
          <p>
            The following is a non-exclusive list of conduct for which we may,
            in our sole discretion, ban you:
          </p>
          <ol>
            <li>
              <p>
                <span className="bullet-heading">Cheating.</span>
                We consider all the following to be instances of cheating:
              </p>
              <ul>
                <li> Using a computer engine to assist your play</li>
                <li>
                  Using word lookup programs (including but not limited to
                  Zyzzyva, Xerafin, and Ulu) to assist your play
                </li>
                <li>Asking another player for help during a game.</li>
                <li>
                  Using online broadcasts/commentary of your game in progress to
                  assist your play
                </li>
                <li>
                  Observing your own game on Woogles.io with a second account
                </li>
              </ul>
              <p>
                This list is not exhaustive and we reserve the right to
                determine what we consider to be cheating on Woogles.io.
                Woogles.io uses advanced algorithms to detect instances of
                cheating. Woogles.io is not required to disclose and will never
                disclose the precise methods used in its cheating detection
                algorithms.
              </p>
              <p>
                We will not consider the above behavior cheating if and only if
                you are playing in a nonrated setting and receive explicit
                permission from your opponent to do so. For instance, we
                encourage consultation games as long as they are unrated.
              </p>
              <p>
                It is permissible to check the validity of words while playing a
                game with auto-adjudication (VOID challenge setting).
              </p>
            </li>
            <li>
              <p>
                <span className="bullet-heading">
                  Artificially inflating or deflating your rating.
                </span>
                This is where a User purposefully loses, or has arranged with an
                opponent to win. As a result, the User’s rating will
                artificially increase or decrease. This compromises the
                integrity of our rating system and as such is strictly
                prohibited.
              </p>
            </li>
            <li>
              <p>
                <span className="bullet-heading">Impersonation.</span>
                Don’t pretend to be someone who you’re not.
              </p>
            </li>
            <li>
              <p>
                <span className="bullet-heading">
                  Attempting to break into accounts of other users, as well as
                  DDoS and volumetric attacks.
                </span>
              </p>
            </li>
            <li>
              <p>
                <span className="bullet-heading">
                  Harassing conduct or offensive language.
                </span>
                We reserve the right for our team and mods to unilaterally
                determine what constitutes harassment or offensive language.
              </p>
            </li>
            <li>
              <p>
                <span className="bullet-heading">
                  Sharing copyrighted material,{' '}
                </span>
                defined as posting anything other than what you’ve made
                yourself, or have a fiduciary right to share. Users are
                responsible for what they post and share.
              </p>
            </li>
            <li>
              <p>
                <span className="bullet-heading">
                  Abuse our infrastructure.
                </span>
                Don’t take any action that imposes an unreasonable load on our
                infrastructure or on our third-party providers. We reserve the
                right to determine what is reasonable.
              </p>
            </li>
            <li>
              <p>
                <span className="bullet-heading">Spamming</span>
                Do not post unsolicited messages like junk mail, chain letters,
                or general spam.
              </p>
            </li>
          </ol>
          <p>
            This list is non-exhaustive and doesn’t detail all activity we can
            or will take action against. Egregious misconduct may, in our sole
            discretion, also be reported to relevant criminal authorities when
            we believe warranted. The final word is always with the
            administrators, and penalties will be applied at the administrators’
            discretion.
          </p>
          <h4>Closing Your Account</h4>
          <p>
            You can close your account at any time. Closing your account will
            immediately and permanently erase most of your account and all
            content and data. You can close your account through your settings.
          </p>
          <p>
            Data that is not erased upon account closure: We will not delete the
            existence of games in which you participated; however, your username
            will no longer be associated with those games. We may also retain
            certain information as required by law or as necessary for our
            legitimate operating purposes.
          </p>
          <p>
            All of the provisions within this agreement survive the closure of
            an account.
          </p>
          <h4>Our Rights</h4>
          <p>
            To operate, we need to be able to maintain control over what happens
            on our website. To summarize, we reserve the right to make decisions
            to protect the health and integrity of our systems. We’ll only use
            these powers when we absolutely must.
          </p>
          <p>
            We can make changes to the Woogles Site and Services without letting
            you know in advance and without being liable for any loss, damage,
            or harm that arises as a result.
          </p>
          <p>
            We have the ultimate say in who can use our Site and Services. We
            can cancel accounts, or decline to let Users use our Services.
          </p>
          <p>
            We are not responsible for any loss, damage, or harm that may arise
            from any occasional downtime.
          </p>
          <p>
            We reserve the right to contact the relevant authorities if we have
            reasonable belief that a User is or may be breaking the law, and to
            share relevant data and information with appropriate authorities,
            for any reason, at our sole discretion.
          </p>
          <p>
            We will aggregate all chat logs from our lobby help chat, tournament
            rooms, and private chat, and reserve the right to read through chat
            logs for evidence of misconduct according to our Fairplay and
            Community guidelines above.
          </p>
          <p>
            This list is non-exhaustive, and we reserve the right to add, edit,
            or remove entries from it at any time.
          </p>
          <h4>Privacy</h4>
          <p>
            Through your use of our Service, you consent to the collection and
            use of your data. Woogles will never share your data with third
            party trackers such as Google Analytics.
          </p>
          <h4>Copyright</h4>
          <p>
            Woogles is licensed under the
            <a
              href="https://github.com/domino14/liwords/blob/master/LICENSE"
              target="_blank"
              rel="noopener noreferrer"
            >
              {' '}
              GNU AGPL license.
            </a>
          </p>
          <p>
            Woogles respects the intellectual property rights of others and
            expects Users of the Services to do the same. Woogles reserves the
            right to remove content alleged to be infringing without prior
            notice at our sole discretion and without liability to you. We will
            respond to notices of alleged copyright infringement. If you believe
            your content has been copied in a way that constitutes copyright
            infringement, please alert us by contacting us at woogles at
            woogles.io.
          </p>
          <p>
            You retain your rights to any content you submit. By submitting,
            posting or displaying content on or through our services, you grant
            us a global, non-exclusive, royalty-free license, with the right to
            sublicense, to use, copy, reproduce, process, adapt, modify,
            publish, transmit, display and distribute such content in any and
            all media or distribution channels. Such additional uses may be made
            without compensation paid to you with respect to the content you
            make available through the services. Woogles does not control how
            third parties make use of your content.
          </p>
          <h4>Donations</h4>
          <p>
            Any user may use Woogles.io for free in perpetuity, with no
            expectation of payment. Users can donate to Woogles Inc via the
            donation link on our homepage (
            <a href="/donate">woogles.io/donate</a>), or by becoming a
            <a
              href="https://www.patreon.com/woogles_io/"
              target="_blank"
              rel="noopener noreferrer"
            >
              {' '}
              Patreon backer
            </a>
            . Woogles will receive all funds donated except where there are
            third party transaction fees. Your donations to Woogles are
            charitable gifts in the 501(c)(3) sense, and therefore
            tax-deductible - please reach out to woogles at woogles.io to
            request a donation receipt.
          </p>
          <p>
            Donations are gifts to Woogles, and do not represent investments or
            loans, nor do they imply any additional transfer of services. We
            will refund donations if requested.
          </p>
          <h4>Stuff we’re not responsible for</h4>
          <p>
            Services made available are on an as-is and as-available basis.
            Woogles makes no warranty or representation and disclaims all
            responsibility and liability for the completeness, availability,
            timeliness, accuracy and security of the content and services.
          </p>
          <p>
            We disclaim all responsibility for any harm or loss of any data to
            your computer system, the deletion of or failure to store or
            transmit communications or content maintained by the services or
            whether the services will meet your requirements or be available on
            an uninterrupted, secure or error-free basis. Woogles cannot be held
            responsible for any loss, financial or otherwise, from third party
            sources, including misrepresentations, negligence or fraud.
          </p>
          <p>
            No advice, information or written correspondence whether oral or
            written procured from Woogles entities or through Woogles services
            will create any warranty or representation not expressly made
            herein.
          </p>
          <h4>Advertisement Policy</h4>
          <p>
            Woogles will never knowingly host advertisements. Woogles is not
            responsible for third party advertisements, widgets and apps hosted
            by service providers on Woogles social media.
          </p>
          <h4>Jurisdiction and Dispute Resolution</h4>
          <p>
            In the event of any dispute relating to these Terms or by using our
            Services, we encourage you to contact our team first at woogles at
            woogles.io such that we can attempt to work out an amicable
            resolution.
          </p>
          <p>
            You agree to accept the law and jurisdiction of the State of New
            Jersey by using our services, without giving effect to any principle
            of conflict of law. Any legal suit, action or proceeding arising out
            of, or related to, these Terms of Service shall be filed only in the
            state or federal courts located in Essex County, New Jersey, and you
            hereby consent and submit to the venue and personal jurisdiction of
            such courts for the purposes of such action.
          </p>
          <p>
            In the event any provision of these terms is held to be invalid or
            unenforceable, then that provision will be limited to the minimum
            enforcement possible, and the remaining terms held with full effect.
          </p>
          <p>These terms are an agreement between you and Woogles.</p>
        </main>
      </div>
    </>
  );
};
