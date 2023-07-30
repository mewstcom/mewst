# typed: strict
# frozen_string_literal: true

class Resources::Internal::Base
  extend T::Sig

  include Alba::Resource

  class ErrorCode < T::Enum
    enums do
      InvalidInputData = new("invalid_input_data")
      NotFound = new("not_found")
    end
  end

  class Error < T::Struct
    const :code, Resources::Internal::Base::ErrorCode
    const :message, String
  end
end
