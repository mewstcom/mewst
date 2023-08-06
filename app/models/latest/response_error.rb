# typed: strict
# frozen_string_literal: true

class Latest::ResponseError < T::Struct
  const :code, Latest::ResponseErrorCode
  const :field, T.nilable(String)
  const :message, String
end
