# typed: strict
# frozen_string_literal: true

module TextHelper
  extend T::Sig

  sig { params(content: String).returns(String) }
  def render_content(content)
    simple_format(content)
  end
end
