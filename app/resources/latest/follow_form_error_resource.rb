# typed: strict
# frozen_string_literal: true

class Latest::FollowFormErrorResource < Latest::FormErrorResource
  sig { override.returns(T.nilable(String)) }
  def field
    case error.attribute
    when :target_atname
      "atname"
    else
      error.attribute.to_s
    end
  end
end
