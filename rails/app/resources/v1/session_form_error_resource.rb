# typed: strict
# frozen_string_literal: true

class V1::SessionFormErrorResource < V1::FormErrorResource
  sig { override.returns(T.nilable(String)) }
  def field
    case error.attribute
    when :base
      "email_or_password"
    else
      error.attribute.to_s
    end
  end
end
