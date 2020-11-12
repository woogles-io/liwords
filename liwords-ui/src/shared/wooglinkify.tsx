import React from 'react';

export const Wooglinkify = (props: { message: string }) => {
  const { message } = props;

  const rendered = React.useMemo(() => {
    const re = /\bhttps?:\/\/\S+/g;
    let pos = 0;
    const arr = [];
    for (let match; (match = re.exec(message)); ) {
      const { 0: chunk, index } = match;
      if (pos < index) {
        arr.push(
          <React.Fragment key={arr.length}>
            {message.substring(pos, index)}
          </React.Fragment>
        );
      }
      arr.push(
        <a
          key={arr.length}
          target="_blank"
          rel="noopener noreferrer"
          href={chunk}
        >
          {chunk}
        </a>
      );
      pos = index + chunk.length;
    }
    return <React.Fragment>{arr}</React.Fragment>;
  }, [message]);

  return rendered;
};
