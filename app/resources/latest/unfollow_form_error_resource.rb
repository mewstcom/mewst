# typed: strict
# frozen_string_literal: true

class Latest::UnfollowFormErrorResource < Latest::FormErrorResource
  sig { override.returns(T.nilable(String)) }
  def field
    case error.attribute
    when :target_profile
      "atname"
    else
      error.attribute.to_s
    end
  end
end
