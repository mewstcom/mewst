# typed: strict
# frozen_string_literal: true

module TextHelper
  extend T::Sig

  sig { params(content: String).returns(String) }
  def render_content(content)
    # 2回以上の改行は1つの空行にする
    replaced_content = content.gsub(/\n{2,}/, "<br>\n")
    auto_link(simple_format(replaced_content), html: {class: "link link-info", target: "_blank"}, link: :urls)
  end
end
