# typed: strict
# frozen_string_literal: true

class Internal::Resources::Base
  extend T::Sig

  include Alba::Resource

  class ErrorCode < T::Enum
    enums do
      InvalidInputData = new("invalid_input_data")
      NotFound = new("not_found")
    end
  end

  class Error < T::Struct
    const :code, Internal::Resources::Base::ErrorCode
    const :message, String
  end
end
