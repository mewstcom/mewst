# typed: strict
# frozen_string_literal: true

module TextHelper
  extend T::Sig

  sig { params(content: String).returns(String) }
  def render_content(content)
    auto_link(simple_format(content), html: {class: "link link-info", target: "_blank"}, link: :urls)
  end
end
