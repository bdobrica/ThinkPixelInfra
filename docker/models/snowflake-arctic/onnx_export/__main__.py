from pathlib import Path
from typing import Dict

import torch
from optimum.exporters.onnx import export
from optimum.exporters.onnx.model_configs import XLMRobertaOnnxConfig
from transformers import AutoConfig, AutoModel, AutoTokenizer
from transformers.configuration_utils import PretrainedConfig
from transformers.modeling_utils import PreTrainedModel


class ExportConfig(XLMRobertaOnnxConfig):
    @property
    def outputs(self) -> Dict[str, Dict[int, str]]:
        return {
            "embeddings": {
                0: "batch_size",
                1: "sequence_length",
            },
            "attentions": {
                0: "batch_size",
                1: "sequence_length",
            },
        }


class ExportReadyModel(PreTrainedModel):
    def __init__(self, config: PretrainedConfig, model: PreTrainedModel):
        super().__init__(config)
        self._model = model

    def forward(self, input_ids, attention_mask, **kwargs):
        _output = self._model(input_ids, attention_mask, **kwargs)

        output = {}
        if hasattr(_output, "last_hidden_state"):
            output["embeddings"] = torch.nn.functional.normalize(
                _output.last_hidden_state[:, 0],  # (batch, embedding_size)
                p=2,
                dim=1,
            )
        if hasattr(_output, "attentions"):
            output["attentions"] = torch.mean(
                _output.attentions[-1][:, :, 0, :],  # (batch, num_heads, seq_len)
                dim=1,
            )  # (batch, seq_len)
        return output


if __name__ == "__main__":
    model_id = "Snowflake/snowflake-arctic-embed-l-v2.0"
    tokenizer = AutoTokenizer.from_pretrained(model_id)

    model = AutoModel.from_pretrained(model_id)
    config = AutoConfig.from_pretrained(model_id)

    export_config = ExportConfig(config=config, task="feature-extraction")
    export_ready_model = ExportReadyModel(config, model)

    export(
        export_ready_model,
        export_config,
        Path("model.onnx"),
        model_kwargs={"output_attentions": True},
    )
