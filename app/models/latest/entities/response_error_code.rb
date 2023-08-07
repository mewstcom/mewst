# typed: strict
# frozen_string_literal: true

class Latest::Entities::ResponseErrorCode < T::Enum
  enums do
    InvalidInputData = new("invalid_input_data")
    NotFound = new("not_found")
  end
end
