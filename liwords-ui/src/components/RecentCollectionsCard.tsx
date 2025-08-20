import { Button, Card, Table, Tooltip } from "antd";
import moment from "moment";
import React from "react";
import { Link } from "react-router";
import { Collection } from "../gen/api/proto/collections_service/collections_service_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

type Props = {
  collections: Array<Collection>;
  fetchPrev?: () => void;
  fetchNext?: () => void;
  loggedInUserUuid?: string;
};

export const RecentCollectionsCard = React.memo((props: Props) => {
  const formattedCollections = props.collections.map((collection) => {
    const whenMoment = moment(
      collection.updatedAt ? timestampDate(collection.updatedAt) : "",
    );
    const when = (
      <Tooltip title={whenMoment.format("LLL")}>{whenMoment.fromNow()}</Tooltip>
    );

    const isOwner = props.loggedInUserUuid === collection.creatorUuid;

    return {
      collectionId: collection.uuid,
      title: (
        <Link to={`/collections/${collection.uuid}`}>{collection.title}</Link>
      ),
      creator: (
        <Link to={`/profile/${collection.creatorUsername}`}>
          {collection.creatorUsername}
        </Link>
      ),
      games: `${collection.gameCount || 0} game(s)`,
      updatedAt: when,
      actions: (
        <Link to={`/collections/${collection.uuid}`}>
          {isOwner ? "Edit" : "View"}
        </Link>
      ),
    };
  });

  const columns = [
    {
      title: "Collection",
      dataIndex: "title",
      key: "title",
      width: "30%",
    },
    {
      title: "Creator",
      dataIndex: "creator",
      key: "creator",
      width: "20%",
    },
    {
      title: "Games",
      dataIndex: "games",
      key: "games",
      width: "15%",
    },
    {
      title: "Updated",
      dataIndex: "updatedAt",
      key: "updatedAt",
      width: "15%",
    },
    {
      title: "Action",
      dataIndex: "actions",
      key: "actions",
      width: "10%",
    },
  ];

  return (
    <Card
      title="Recently Updated Collections"
      className="game-history-card"
      style={{ marginBottom: "24px" }}
    >
      <Table
        className="game-history"
        columns={columns}
        dataSource={formattedCollections}
        pagination={{
          hideOnSinglePage: true,
          defaultPageSize: Infinity,
        }}
        rowKey="collectionId"
        size="small"
      />
      <div className="game-history-controls">
        <Button disabled={!props.fetchPrev} onClick={props.fetchPrev}>
          Prev
        </Button>
        <Button disabled={!props.fetchNext} onClick={props.fetchNext}>
          Next
        </Button>
      </div>
    </Card>
  );
});
