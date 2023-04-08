# typed: strict
# frozen_string_literal: true

module TextHelper
  extend T::Sig

  sig { params(comment: String).returns(String) }
  def render_comment(comment)
    auto_link(simple_format(comment), html: {target: "_blank"}, link: :urls)
  end
end
