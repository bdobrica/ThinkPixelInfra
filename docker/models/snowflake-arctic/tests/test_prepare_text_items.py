from snowflake_arctic.preprocess import prepare_text_items


# --- prepare_text_items Tests ---
def test_prepare_text_items_long_text():
    text = (
        "Yes, we became very wakeful; so much so that our recumbent position began to grow wearisome, and by "
        "little and little we found ourselves sitting up; the clothes well tucked around us, leaning against the "
        "head-board with our four knees drawn up close together, and our two noses bending over them, as if our "
        "kneepans were warming-pans. We felt very nice and snug, the more so since it was so chilly out of doors; "
        "indeed out of bed-clothes too, seeing that there was no fire in the room. The more so, I say, because "
        "truly to enjoy bodily warmth, some small part of you must be cold, for there is no quality in this world "
        "that is not what it is merely by contrast. Nothing exists in itself. If you flatter yourself that you are "
        "all over comfortable, and have been so a long time, then you cannot be said to be comfortable any more. "
        "But if, like Queequeg and me in the bed, the tip of your nose or the crown of your head be slightly "
        "chilled, why then, indeed, in the general consciousness you feel most delightfully and unmistakably warm. "
        "For this reason a sleeping apartment should never be furnished with a fire, which is one of the luxurious "
        "discomforts of the rich. For the height of this sort of deliciousness is to have nothing but the blanket "
        "between you and your snugness and the cold of the outer air. Then there you lie like the one warm spark "
        "in the heart of an arctic crystal."
    )
    items = [{"text": text, "metadata": {}}]
    batch = prepare_text_items(items, chunk_size=1000, chunk_overlap=200)
    assert len(batch) > 1
    for item in batch:
        assert len(item["text"]) <= 1000


def test_prepare_text_items_short_text():
    text = "The quick brown fox jumps over the lazy dog."
    items = [{"text": text, "metadata": {}}]
    batch = prepare_text_items(items, chunk_size=1000, chunk_overlap=200)
    assert len(batch) == 1
    assert batch[0]["text"] == text


def test_prepare_text_items_metadata():
    text = "metadata test"
    items = [{"text": text, "metadata": {"foo": "bar", "extra": {"baz": 123}}}]
    batch = prepare_text_items(items, chunk_size=1000, chunk_overlap=200)
    assert batch[0]["metadata"]["foo"] == "bar"
    assert batch[0]["metadata"]["extra"]["baz"] == 123
