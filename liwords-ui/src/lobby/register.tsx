import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import { TopBar } from "../navigation/topbar";
import {
  Input,
  Form,
  Button,
  Alert,
  Checkbox,
  Select,
  AutoComplete,
  Modal,
} from "antd";
import { Rule } from "antd/lib/form";
import "./accountForms.scss";
import woogles from "../assets/woogles.png";
import { useLoginStateStoreContext } from "../store/store";
import { LoginModal } from "./login";
import { countryArray } from "../settings/country_map";
import { connectErrorMessage, useClient } from "../utils/hooks/connect";
import {
  AuthenticationService,
  RegistrationService,
} from "../gen/api/proto/user_service/user_service_pb";

const allMonthNames = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];
const allMonthLowercaseNames = allMonthNames.map((name) => name.toLowerCase());
const allMonthNumbers = allMonthNames.map((_, idx) =>
  String(idx + 1).padStart(2, "0"),
);
const allMonthOptions = allMonthNames.map((name, idx) => ({
  value: name,
  label: `${allMonthNumbers[idx]} - ${name}`,
}));

const allDateOptions = Array.from(new Array(31), (_, x) => ({
  value: String(x + 1).padStart(2, "0"),
}));

// 2006-01-02 (hi Golang), 2012-01-02, 2006-10-02, 2006-01-09 are Mondays.
// Returns [0, 1, 2] in some order where 0 = year, 1 = month, 2 = date.
const determineLocalDateSequence = () => {
  const fmt = new Intl.DateTimeFormat();
  const s1 = fmt.format(new Date(2006, 0, 2));
  const dist = (s2: string) => {
    const l = Math.min(s1.length, s2.length);
    for (let i = 0; i < l; ++i) if (s1[i] !== s2[i]) return i;
    return l;
  };
  return [
    dist(fmt.format(new Date(2012, 0, 2))) * 3,
    dist(fmt.format(new Date(2006, 9, 2))) * 3 + 1,
    dist(fmt.format(new Date(2006, 0, 9))) * 3 + 2,
  ]
    .sort((a, b) => (a < b ? -1 : 1))
    .map((x) => x % 3);
};

type Option = {
  label?: string;
  value: string;
};

const useBirthBox = (
  allOptions: Array<Option>,
  deduceSelection: (s: string) => number | undefined,
  formatSelection: (n: number) => string,
  filterOptions: (s: string) => Array<Option>,
  placeholder: string,
) => {
  const [selection, setSelection] = useState<number | undefined>(undefined);
  const [searched, setSearched] = useState("");
  const [shownOptions, setShownOptions] = useState(allOptions);
  useEffect(() => {
    setShownOptions(allOptions);
  }, [allOptions]);
  const handleChange = useCallback(
    (s: string) => {
      const selection = deduceSelection(s);
      setSelection(selection);
      if (selection != null) {
        setSearched(formatSelection(selection));
      }
      setShownOptions(allOptions);
    },
    [deduceSelection, formatSelection, allOptions],
  );
  const handleSearch = useCallback(
    (s: string) => {
      setSearched(s);
      setShownOptions(filterOptions(s));
    },
    [filterOptions],
  );
  const [reformatSelection, setReformatSelection] = useState<boolean>(false);
  const handleDropdownVisibleChange = useCallback((open: boolean) => {
    if (!open) {
      // the reformatting has to be in the next render,
      // otherwise it would revert to the current value instead.
      setReformatSelection(true);
    }
  }, []);
  useEffect(() => {
    if (reformatSelection) {
      setReformatSelection(false);
      if (selection != null) {
        setSearched(formatSelection(selection));
      }
      setShownOptions(allOptions);
    }
  }, [reformatSelection, selection, formatSelection, allOptions]);

  return [
    <Form.Item key="stupidlinter">
      <AutoComplete
        value={searched}
        options={shownOptions}
        onChange={handleChange}
        onSearch={handleSearch}
        onDropdownVisibleChange={handleDropdownVisibleChange}
        popupClassName="birthdate-dropdown"
      >
        <Input variant="borderless" {...{ placeholder }} />
      </AutoComplete>
    </Form.Item>,
    selection,
  ];
};

const usernameValidator = async (rule: Rule, value: string) => {
  if (!value) {
    throw new Error("Please input your username");
  }
  if (value.length < 3) {
    throw new Error("Min username length is 3");
  }
  if (value.length > 20) {
    throw new Error("Max username length is 20");
  }
  if (!/^[0-9a-zA-Z\-_.]+$/.test(value)) {
    throw new Error("Valid characters are A-Z a-z 0-9 . _ -");
  }
  if (!/^[0-9a-zA-Z]/.test(value)) {
    throw new Error("Valid starting characters are A-Z a-z 0-9");
  }
  if (!/[0-9a-zA-Z]$/.test(value)) {
    throw new Error("Valid ending characters are A-Z a-z 0-9");
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
      const d = new Date(Date.UTC(iy, im, id));
      if (
        d.getUTCFullYear() === iy &&
        d.getUTCMonth() === im &&
        d.getUTCDate() === id &&
        // ignore any timezone-induced off-by-1-day, they're too young anyway
        +d <= Date.now()
      ) {
        valid = true;
      }
    }
  }
  if (!valid) {
    throw new Error("Please input your birth date");
  }
};

export const Register = () => {
  const [err, setErr] = useState("");
  const [signedUp, setSignedUp] = useState(false);

  const [loginModalVisible, setLoginModalVisible] = useState(false);
  const { loginState } = useLoginStateStoreContext();
  const handleShowLoginModal = useCallback(
    (evt: React.MouseEvent<HTMLElement>) => {
      evt.preventDefault();
      setLoginModalVisible(true);
    },
    [],
  );
  const authClient = useClient(AuthenticationService);
  const registrationClient = useClient(RegistrationService);

  const onFinish = async (values: { [key: string]: string }) => {
    try {
      await registrationClient.register({
        username: values.username,
        password: values.password,
        email: values.email,
        registrationCode: values.registrationCode,
        birthDate: values.birthDate,
        firstName: values.firstName,
        lastName: values.lastName,
        countryCode: values.countryCode,
      });

      // Try to login - this will succeed if email verification is disabled (dev mode)
      // or fail if email verification is required
      try {
        await authClient.login({
          username: values.username,
          password: values.password,
        });
        setSignedUp(true);
      } catch (loginError) {
        const loginErrorMsg = connectErrorMessage(loginError);

        // Check if the error is due to email verification
        if (loginErrorMsg.toLowerCase().includes("verify your email")) {
          Modal.success({
            title: "Registration Successful!",
            content: (
              <div>
                <p>
                  Thank you for signing up! We've sent a verification email to{" "}
                  <strong>{values.email}</strong>.
                </p>
                <p>
                  Please check your inbox (and spam folder) and click the
                  verification link to activate your account.
                </p>
                <p>The link will expire in 48 hours.</p>
              </div>
            ),
            okText: "Got it",
            onOk: () => {
              navigate("/", { replace: true });
            },
          });
        } else {
          // Some other login error occurred
          setErr(loginErrorMsg);
        }
      }
    } catch (e) {
      // Registration itself failed
      setErr(connectErrorMessage(e));
    }
  };

  const navigate = useNavigate();
  const loggedIn = signedUp || loginState.loggedIn;
  useEffect(() => {
    if (loggedIn) {
      navigate("/", { replace: true });
    }
  }, [navigate, loggedIn]);

  // XXX: This is copied from settings/personal_info.tsx (with added placeholder).
  // It has the same issues (such as the emoji not displaying on Windows).
  const countrySelector = (
    <Select
      size="large"
      bordered={false}
      placeholder="Country"
      popupClassName="country-dropdown"
    >
      {countryArray.map((country) => {
        return (
          <Select.Option key={country.code} value={country.code.toLowerCase()}>
            {country.emoji} {country.name}
          </Select.Option>
        );
      })}
    </Select>
  );

  const currentTime = new Date(Date.now());
  const currentYear = currentTime.getFullYear(); // intentionally using browser timezone
  const birthYearOptions = useMemo(() => {
    const a = [];
    for (let i = currentYear; i >= 1900; --i) {
      a.push({ value: String(i).padStart(4, "0") });
    }
    return a;
  }, [currentYear]);

  const [birthYearBox, birthYearSelected] = useBirthBox(
    birthYearOptions,
    useCallback(
      (s) => {
        // use the actual value because the index may shift near start of year
        let y = birthYearOptions.find(({ value }) => value === s);
        if (y == null && /^\d+$/.test(s)) {
          // allow omitting leading zeros
          const i = +s;
          y = birthYearOptions.find(({ value }) => +value === i);
          if (y == null && s.length === 2) {
            // allow typing just two digits
            y = birthYearOptions.find(({ value }) => {
              const ivalue = +value;
              const age = currentYear - ivalue;
              return age >= 0 && age <= 99 && ivalue % 100 === i;
            });
          }
        }
        return y == null ? undefined : +y.value;
      },
      [birthYearOptions, currentYear],
    ),
    useCallback((n) => String(n).padStart(4, "0"), []),
    useCallback(
      (s) => birthYearOptions.filter(({ value }) => value.includes(s)),
      [birthYearOptions],
    ),
    "Year",
  );

  const [birthMonthBox, birthMonthSelected] = useBirthBox(
    allMonthOptions,
    useCallback((s) => {
      let idx = -1;
      if (s.length >= 3) {
        // allow month prefixes
        const lowerSearch = s.toLowerCase();
        const matchingMonths = allMonthLowercaseNames.filter(
          (monthLowercaseName) => monthLowercaseName.startsWith(lowerSearch),
        );
        if (matchingMonths.length === 1) {
          // only one match, it wins
          idx = allMonthLowercaseNames.findIndex((monthLowercaseName) =>
            monthLowercaseName.startsWith(lowerSearch),
          );
        }
      }
      if (idx < 0 && /^\d+$/.test(s)) {
        // allow month numbers and omitting leading zeros
        const i = +s;
        if (i >= 1 && i <= allMonthNames.length) {
          idx = i - 1;
        }
      }
      return idx < 0 ? undefined : idx + 1;
    }, []),
    useCallback((n) => {
      if (n >= 1 && n <= allMonthNames.length) {
        return allMonthNames[n - 1];
      } else {
        return String(n).padStart(2, "0");
      }
    }, []),
    useCallback((s) => {
      const lowerSearch = s.toLowerCase();
      return allMonthOptions.filter(
        (_, idx) =>
          allMonthLowercaseNames[idx].includes(lowerSearch) ||
          allMonthNumbers[idx].includes(lowerSearch),
      );
    }, []),
    "Month",
  );

  const [birthDateBox, birthDateSelected] = useBirthBox(
    allDateOptions,
    useCallback((s) => {
      let idx = allDateOptions.findIndex(({ value }) => value === s);
      if (idx < 0 && /^\d+$/.test(s)) {
        // allow omitting leading zeros
        const i = +s;
        idx = allDateOptions.findIndex(({ value }) => +value === i);
      }
      return idx < 0 ? undefined : idx + 1;
    }, []),
    useCallback((n) => String(n).padStart(2, "0"), []),
    useCallback(
      (s) => allDateOptions.filter(({ value }) => value.includes(s)),
      [],
    ),
    "Date",
  );

  const localDateSequence = useMemo(() => determineLocalDateSequence(), []);
  const birthBoxes = [
    <span className="birth-year" key="birth-year">
      {birthYearBox}
    </span>,
    <span className="birth-month" key="birth-month">
      {birthMonthBox}
    </span>,
    <span className="birth-date" key="birth-date">
      {birthDateBox}
    </span>,
  ];

  const [form] = Form.useForm();
  useEffect(() => {
    const birthDate =
      birthYearSelected != null &&
      birthMonthSelected != null &&
      birthDateSelected != null
        ? `${String(birthYearSelected).padStart(4, "0")}-${String(
            birthMonthSelected,
          ).padStart(2, "0")}-${String(birthDateSelected).padStart(2, "0")}`
        : "";
    const oldValue = form.getFieldValue("birthDate") ?? "";
    if (oldValue !== birthDate) {
      form.setFieldsValue({ birthDate });
      if (birthDate) {
        // validate only when complete, don't nag "required" while filling in.
        // for example this would show an error when selecting 31 Feb.
        form.validateFields(["birthDate"]); // async
      }
    }
  }, [form, birthYearSelected, birthMonthSelected, birthDateSelected]);

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
            friendly match you’re in the right place.
          </p>
          <p>- Woogles and team</p>
          <Form layout="inline" name="register" onFinish={onFinish} form={form}>
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
                  type: "email",
                  message: "This is not a valid email",
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
                  message: "Please input your password",
                },
                {
                  min: 8,
                  message: "Password should be at least 8 characters",
                },
              ]}
            >
              <Input.Password
                placeholder="Password"
                autoComplete="new-password"
              />
            </Form.Item>

            <p className="persistent-explanation">
              Password should be at least 8 characters.
              {/*
              Password must include at least one lowercase letter, upper case
              letter, number, and symbol.
              */}
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

            {localDateSequence.map((k) => (
              <React.Fragment key={k}>{birthBoxes[k]}</React.Fragment>
            ))}
            <span className="full-width only-errors">
              <Form.Item
                name="birthDate"
                rules={[
                  {
                    validator: birthDateValidator,
                  },
                ]}
              ></Form.Item>
            </span>

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
                    message: "You must agree to this condition",
                    transform: (value) => value || undefined,
                    type: "boolean",
                  },
                ]}
                valuePropName="checked"
                initialValue={false}
                name="nocheating"
              >
                <Checkbox>
                  <p className="no-cheat">
                    I agree to the{" "}
                    <Link target="_blank" to="/terms">
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
          {err !== "" ? <Alert message={err} type="error" /> : null}

          <p>
            Already have a Woogles account? No worries!{" "}
            <Link to="/" onClick={handleShowLoginModal}>
              Log in here
            </Link>
            .
            <LoginModal {...{ loginModalVisible, setLoginModalVisible }} />
          </p>
        </div>
      </div>
    </>
  );
};
