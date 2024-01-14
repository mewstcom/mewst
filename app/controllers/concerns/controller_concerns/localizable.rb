# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Localizable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { params(action: Proc).returns(T.untyped) }
  def set_locale(&action)
    I18n.with_locale(current_locale.serialize, &action)
  end

  sig { returns(Locale) }
  private def preferred_locale
    preferred_languages = http_accept_language.user_preferred_languages
    # Chrome returns "ja", but Safari would return "ja-JP", not "ja".
    (preferred_languages.any? { |lang| lang.match?(/ja/) }) ? Locale::Ja : default_locale
  end

  sig { returns(Locale) }
  private def default_locale
    Locale::En
  end

  sig { returns(Locale) }
  private def current_locale
    current_viewer&.locale.presence || preferred_locale
  end
end
