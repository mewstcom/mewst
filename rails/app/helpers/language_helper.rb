# typed: strict
# frozen_string_literal: true

module LanguageHelper
  extend T::Sig

  sig { params(lang: Symbol).returns(String) }
  def lang_option_text(lang:)
    text = t("nouns.#{lang}")
    text_native = t("nouns.#{lang}_native")

    if text == text_native
      text
    else
      "#{text} - #{text_native}"
    end
  end
end
