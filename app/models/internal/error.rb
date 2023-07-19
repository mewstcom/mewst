# typed: strict
# frozen_string_literal: true

class Internal::Error < T::Struct
  const :code, Internal::ErrorCode
  const :message, String
end
