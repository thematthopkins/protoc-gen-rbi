RSpec.describe "Mytest", :type => :request do
  let(:filled_in_json) do
      <<-JSON
      {
  "s": {
    "int32Field": 11
  },
  "ss": [
    {
      "int32Field": 111
    },
    {
      "int32Field": 222
    }
  ],
  "colour": "RED",
  "colours": [
    "RED",
    "RED"
  ],
  "intField": 123,
  "intFields": [
    111,
    222,
    333
  ],
  "int64Field": "665544",
  "int64Fields": [
    "55"
  ],
  "timestampField": "2023-11-20T13:07:34Z",
  "timestampFields": [
    "2023-11-20T13:07:34Z"
  ],
  "oo1": 111,
  "optionalS": {
    "int32Field": 11
  },
  "optionalColour": "RED",
  "optionalIntField": 123,
  "optionalInt64Field": "55",
  "optionalTimestampField": "2023-11-20T13:07:34Z"
}
JSON
  end

  let(:filled_in_val) do
    SimplePb::Foo.new(
      s: SimplePb::Simple.new(int32_field: 11),
      ss: [
        SimplePb::Simple.new(int32_field: 111),
        SimplePb::Simple.new(int32_field: 222)
      ],
      optional_s: SimplePb::Simple.new(int32_field: 11),
      colour: SimplePb::Colour::Red,
      colours: [SimplePb::Colour::Red,SimplePb::Colour::Red],
      optional_colour: SimplePb::Colour::Red,
      int_field: 123,
      int_fields: [111, 222, 333],
      optional_int_field: 123,
      int64_field: 665544,
      int64_fields: [55],
      optional_int64_field: 55, 
      timestamp_field: Time.iso8601("2023-11-20T13:07:34Z"),
      timestamp_fields: [Time.iso8601("2023-11-20T13:07:34Z")],
      optional_timestamp_field: Time.iso8601("2023-11-20T13:07:34Z"),
      oo: SimplePb::Foo::Oo::Oo1.new(111)
    )
  end

  let(:default_json) do
    <<-JSON
{
  "s": {},
  "oo1": 0
}
    JSON
  end

  let(:default_val) do
    SimplePb::Foo.new(
      s: SimplePb::Simple.new(int32_field: 0),
      ss: [],
      optional_s: nil,
      colour: SimplePb::Colour::Red,
      colours: [],
      optional_colour: nil,
      int_field: 0,
      int_fields: [],
      optional_int_field: nil,
      int64_field: 0,
      int64_fields: [],
      optional_int64_field: nil, 
      timestamp_field: Time.iso8601("1970-01-01T00:00:00Z"),
      timestamp_fields: [],
      optional_timestamp_field: nil,
      oo: SimplePb::Foo::Oo::Oo1.new(0)
    )
  end

  let(:default_json_out) do
    <<-JSON
{
  "colour" : "RED",
  "colours" : [],
  "int64Field" : "0",
  "int64Fields" : [],
  "intField" : 0,
  "intFields" : [],
  "oo1" : 0,
  "optionalColour" : null,
  "optionalInt64Field" : null,
  "optionalIntField" : null,
  "optionalS" : null,
  "optionalTimestampField" : null,
  "s" : {"int32Field": 0},
  "ss" : [],
  "timestampField" : "1970-01-01T00:00:00Z",
  "timestampFields" : []
}
    JSON
  end

  describe "check smth" do
    it "should match elm's deconding" do
      expect(SimplePb::Foo.decode_json(filled_in_json)).to eq(
        filled_in_val
      )
    end

    it "should match elm's encoding" do
      expect(JSON.parse(filled_in_val.encode_json)).to eq(
        JSON.parse(filled_in_json)
      )
    end
  end

  describe "defaults" do
    it "should match elm's deconding" do
      expect(SimplePb::Foo.decode_json(default_json)).to eq(
        default_val
      )
    end

    it "should match elm's encoding" do
      expect(JSON.parse(default_val.encode_json)).to eq(
        JSON.parse(default_json_out)
      )
    end
  end
end
