# typed: strict
# frozen_string_literal: true

class V1::PostFormErrorResource < V1::FormErrorResource
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
