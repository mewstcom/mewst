# typed: strict
# frozen_string_literal: true

class Latest::Entities::StampFormError < Latest::Entities::FormError
  sig { override.returns(T.nilable(String)) }
  def field
    case error.attribute
    when :target_post
      "post_id"
    else
      error.attribute.to_s
    end
  end
end
