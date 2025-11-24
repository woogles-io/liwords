import React, { useCallback, useEffect, useState } from "react";
import { Button, Slider } from "antd";
import { Modal } from "../utils/focus_modal";
import Cropper, { Area } from "react-easy-crop";

type Props = {
  file?: Blob;
  onCancel: () => void;
  onSave: (imageDataUrl: string) => void;
  onError: (errorMessage: string) => void;
};

export const AvatarCropper = React.memo((props: Props) => {
  const [dataUrl, setDataUrl] = useState<string | undefined>(undefined);
  const [crop, setCrop] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState<number | undefined>(1);
  const [croppedArea, setCroppedArea] = useState<undefined | Area>(undefined);
  const { file, onSave } = props;
  const handleOnSave = useCallback(() => {
    if (!croppedArea || !dataUrl) {
      return;
    }
    const image = new Image();
    image.onload = () => {
      const canvas = document.createElement("canvas"),
        ctx = canvas.getContext("2d");
      canvas.width = 96;
      canvas.height = 96;
      ctx?.drawImage(
        image,
        croppedArea.x,
        croppedArea.y,
        croppedArea.width,
        croppedArea.height,
        0,
        0,
        canvas.width,
        canvas.height,
      );
      const newDataUrl = canvas.toDataURL("image/jpeg", 1);
      onSave(newDataUrl);
    };
    image.src = dataUrl;
  }, [onSave, dataUrl, croppedArea]);

  const onCropComplete = useCallback(
    (croppedArea: Area, croppedAreaPixels: Area) => {
      setCroppedArea(croppedAreaPixels);
    },
    [],
  );

  useEffect(() => {
    const reader = new FileReader();
    reader.onload = () => {
      const image = new Image();
      image.onload = () => {
        const canvas = document.createElement("canvas"),
          width = image.width,
          height = image.height,
          ctx = canvas.getContext("2d");

        canvas.width = width;
        canvas.height = height;
        if (ctx) {
          ctx.fillStyle = "rgba(255,255,255,1)";
          ctx.fillRect(0, 0, width, height);
          ctx.drawImage(image, 0, 0, width, height);
        }
        setDataUrl(canvas.toDataURL("image/jpeg", 1));
      };
      image.src = String(reader.result);
    };
    if (file) {
      reader.readAsDataURL(file);
    }
  }, [file]);

  return (
    <Modal
      className="cropper-modal"
      title="Edit your profile picture"
      maskClosable={false}
      onCancel={props.onCancel}
      open={!!dataUrl}
      width={328}
      footer={
        <>
          <button className="link" onClick={props.onCancel}>
            Cancel
          </button>
          <Button key="submit" type="primary" onClick={handleOnSave}>
            Save
          </Button>
        </>
      }
    >
      <div className="cropper">
        <Cropper
          aspect={1}
          crop={crop}
          cropShape="round"
          image={dataUrl}
          onCropChange={(crop) => {
            setCrop(crop);
          }}
          onCropComplete={onCropComplete}
          zoom={zoom}
          maxZoom={5}
          onZoomChange={(zoom) => {
            setZoom(zoom);
          }}
        />
      </div>
      <div className="zoom-slider">
        -
        <Slider
          tooltip={{ formatter: null }}
          min={1}
          max={5}
          step={0.01}
          value={zoom}
          onChange={(zoom: number) => {
            setZoom(zoom);
          }}
        />
        +
      </div>
    </Modal>
  );
});
