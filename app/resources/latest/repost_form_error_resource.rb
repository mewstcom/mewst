# typed: strict
# frozen_string_literal: true

class Latest::RepostFormErrorResource < Latest::FormErrorResource
  sig { override.returns(T.nilable(String)) }
  def field
    case error.attribute
    when :target_post
      "post_id"
    when :follow
      nil
    else
      error.attribute.to_s
    end
  end
end
