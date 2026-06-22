import React from "react";
import { Card, Col, Row, Space, Tag, Typography } from "antd";
import { Link } from "react-router";
import { LinkOutlined } from "@ant-design/icons";
import type { BroadcastGameStat } from "../../gen/api/proto/broadcast_service/broadcast_service_pb";

const { Text } = Typography;

type Stat = BroadcastGameStat;

function gameLabel(s: Stat): string {
  return `${s.player1Name} vs ${s.player2Name} (R${s.round})`;
}

type CategoryEntry = {
  stat: Stat;
  value: number | string;
  extra?: string;
};

type Category = {
  emoji: string;
  title: string;
  entries: CategoryEntry[];
};

function topN(
  stats: Stat[],
  n: number,
  score: (s: Stat) => number,
  format: (s: Stat, v: number) => string,
  extra?: (s: Stat) => string,
): CategoryEntry[] {
  return stats
    .map((s) => ({ stat: s, raw: score(s) }))
    .sort((a, b) => b.raw - a.raw)
    .slice(0, n)
    .map(({ stat, raw }) => ({
      stat,
      value: format(stat, raw),
      extra: extra?.(stat),
    }));
}

function buildCategories(stats: Stat[]): Category[] {
  const done = stats.filter((s) => s.completedAt);

  // Round leaders: best score in each round (winner's score or max of two)
  const roundLeaders: CategoryEntry[] = (() => {
    const byRound = new Map<number, Stat[]>();
    for (const s of done) {
      const arr = byRound.get(s.round) ?? [];
      arr.push(s);
      byRound.set(s.round, arr);
    }
    const entries: CategoryEntry[] = [];
    for (const [round, games] of Array.from(byRound.entries()).sort(
      ([a], [b]) => a - b,
    )) {
      const top = games.reduce((best, g) => {
        const sc = Math.max(g.player1Score, g.player2Score);
        return sc > Math.max(best.player1Score, best.player2Score) ? g : best;
      });
      const topScore = Math.max(top.player1Score, top.player2Score);
      const topPlayer =
        top.player1Score >= top.player2Score
          ? top.player1Name
          : top.player2Name;
      entries.push({
        stat: top,
        value: topScore,
        extra: `R${round}: ${topPlayer}`,
      });
    }
    return entries.slice(0, 5);
  })();

  return [
    {
      emoji: "🔥",
      title: "Highest Combined Score",
      entries: topN(
        done,
        5,
        (s) => s.player1Score + s.player2Score,
        (s, v) => String(v),
        (s) => `${s.player1Score}–${s.player2Score}`,
      ),
    },
    {
      emoji: "⚔️",
      title: "Closest Games",
      entries: topN(
        done,
        5,
        (s) => -Math.abs(s.player1Score - s.player2Score),
        (s) => `±${Math.abs(s.player1Score - s.player2Score)}`,
        (s) => `${s.player1Score}–${s.player2Score}`,
      ),
    },
    {
      emoji: "🎯",
      title: "Walk-Off Bingos",
      entries: done
        .filter((s) => s.walkOffBingo)
        .slice(0, 5)
        .map((s) => ({
          stat: s,
          value: `${s.player1Score}–${s.player2Score}`,
          extra:
            s.winner === 0 ? `${s.player1Name} won` : `${s.player2Name} won`,
        })),
    },
    {
      emoji: "💥",
      title: "Most Bingos",
      entries: topN(
        done,
        5,
        (s) => s.player1Bingos + s.player2Bingos,
        (_, v) => `${v} bingos`,
        (s) => `${s.player1Bingos}+${s.player2Bingos}`,
      ),
    },
    {
      emoji: "🧊",
      title: "Defensive Masterpieces",
      entries: topN(
        done.filter((s) => s.player1Score !== s.player2Score),
        5,
        (s) => -Math.min(s.player1Score, s.player2Score),
        (s) => `${Math.min(s.player1Score, s.player2Score)} pts allowed`,
        (s) => `${s.player1Score}–${s.player2Score}`,
      ),
    },
    {
      emoji: "⏱",
      title: "Longest Games (moves)",
      entries: topN(
        done,
        5,
        (s) => s.moveCount,
        (_, v) => `${v} moves`,
        (s) => `${s.player1Score}–${s.player2Score}`,
      ),
    },
    {
      emoji: "🏆",
      title: "Upsets",
      entries: (() => {
        const upsets = done
          .filter((s) => {
            if (s.player1Rating === 0 || s.player2Rating === 0) return false;
            const p1Won = s.winner === 0;
            const p2Won = s.winner === 1;
            return (
              (p1Won && s.player1Rating < s.player2Rating) ||
              (p2Won && s.player2Rating < s.player1Rating)
            );
          })
          .map((s) => {
            const winnerRating =
              s.winner === 0 ? s.player1Rating : s.player2Rating;
            const loserRating =
              s.winner === 0 ? s.player2Rating : s.player1Rating;
            const winnerName = s.winner === 0 ? s.player1Name : s.player2Name;
            return {
              stat: s,
              delta: loserRating - winnerRating,
              winnerName,
              winnerRating,
              loserRating,
            };
          })
          .sort((a, b) => b.delta - a.delta)
          .slice(0, 5);
        return upsets.map(
          ({ stat, delta, winnerName, winnerRating, loserRating }) => ({
            stat,
            value: `+${delta} rating diff`,
            extra: `${winnerName} (${winnerRating}) beat ${loserRating}`,
          }),
        );
      })(),
    },
    {
      emoji: "📈",
      title: "Round Leaders",
      entries: roundLeaders,
    },
  ];
}

type CategoryCardProps = {
  category: Category;
};

const CategoryCard: React.FC<CategoryCardProps> = ({ category }) => (
  <Card
    size="small"
    title={
      <span>
        {category.emoji} {category.title}
      </span>
    }
    style={{ marginBottom: 16 }}
  >
    {category.entries.length === 0 ? (
      <Text type="secondary">No data yet.</Text>
    ) : (
      <Space direction="vertical" size={4} style={{ width: "100%" }}>
        {category.entries.map((e, i) => (
          <div
            key={e.stat.gameUuid + i}
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              gap: 8,
            }}
          >
            <div style={{ minWidth: 0, flex: 1 }}>
              <Text strong style={{ fontSize: 13 }}>
                {typeof e.value === "number" ? e.value : e.value}
              </Text>
              {e.extra && (
                <Text type="secondary" style={{ fontSize: 11, marginLeft: 6 }}>
                  {e.extra}
                </Text>
              )}
              <div
                style={{
                  fontSize: 12,
                  color: "#888",
                  whiteSpace: "nowrap",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                }}
              >
                {gameLabel(e.stat)}
              </div>
            </div>
            {e.stat.gameUuid && (
              <Tag>
                <Link to={`/anno/${e.stat.gameUuid}`}>
                  <LinkOutlined /> Review
                </Link>
              </Tag>
            )}
          </div>
        ))}
      </Space>
    )}
  </Card>
);

type Props = {
  stats: BroadcastGameStat[];
};

export const HighlightsTab: React.FC<Props> = ({ stats }) => {
  const categories = buildCategories(stats);

  const left = categories.filter((_, i) => i % 2 === 0);
  const right = categories.filter((_, i) => i % 2 === 1);

  return (
    <div style={{ marginTop: 16 }}>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          {left.map((cat) => (
            <CategoryCard key={cat.title} category={cat} />
          ))}
        </Col>
        <Col xs={24} md={12}>
          {right.map((cat) => (
            <CategoryCard key={cat.title} category={cat} />
          ))}
        </Col>
      </Row>
    </div>
  );
};
