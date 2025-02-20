import React, { useEffect, useState } from "react";
import { useBriefProfile } from "../utils/brief_profiles";
import { ConfigProvider, Tooltip } from "antd";
import { getBadgesMetadata } from "../gen/api/proto/user_service/user_service-ProfileService_connectquery";
import { useQuery } from "@connectrpc/connect-query";

const imageCache: Record<string, string> = {}; // Local cache

const getImage = async (badgeCode: string): Promise<string | null> => {
  try {
    const name = badgeCode.toLowerCase().replaceAll(" ", "_");

    // Check if the image is already cached
    if (imageCache[name]) {
      return imageCache[name]; // Return cached value
    }

    // Dynamically import the image
    const imageModule = await import(`../assets/badges/${name}.png`);
    const imageUrl = imageModule.default;

    // Store in cache
    imageCache[name] = imageUrl;

    return imageUrl;
  } catch (error) {
    console.error(`Failed to load image: ${badgeCode}`, error);
    return null;
  }
};

interface BadgeProps {
  name: string;
  width: number;
}

export const Badge: React.FC<BadgeProps> = ({ name, width }) => {
  const [imgSrc, setImgSrc] = useState<string | null>(null);

  useEffect(() => {
    getImage(name).then(setImgSrc);
  }, [name]);

  if (!imgSrc) return <p>Loading...</p>;

  return <img src={imgSrc} alt={name} width={width} />;
};

interface DisplayUserBadgeProps {
  uuid?: string;
}

export const DisplayUserBadges: React.FC<DisplayUserBadgeProps> = ({
  uuid,
}) => {
  const briefProfile = useBriefProfile(uuid);
  const { data: badgeMetadata } = useQuery(getBadgesMetadata);
  return (
    <ConfigProvider
      theme={{
        components: {
          Tooltip: {
            colorBgSpotlight: "rgba(0, 0, 0, 0)",
            boxShadowSecondary: "none",
          },
        },
      }}
    >
      {briefProfile &&
        briefProfile.badgeCodes.map((bc) => (
          <Tooltip
            key={`${uuid}_badge_${bc}`}
            title={
              <div className="userbadge-tooltip">
                <Badge width={40} name={bc} />
                <div
                  style={{ width: 272, flexShrink: 0, alignSelf: "stretch" }}
                >
                  {badgeMetadata?.badges[bc]}
                </div>
              </div>
            }
          >
            <span style={{ marginLeft: 8 }}>
              <Badge width={20} name={bc} />
            </span>
          </Tooltip>
        ))}
    </ConfigProvider>
  );
};
