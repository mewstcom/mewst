# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Localizable
  extend T::Sig
  extend ActiveSupport::Concern

  sig(:final) { params(action: Proc).returns(T.untyped) }
  def set_locale(&action)
    I18n.with_locale(current_locale.serialize, &action)
  end

  sig(:final) { returns(T.nilable(Locale)) }
  private def instant_locale
    @instant_locale ||= T.let(Locale.try_deserialize(params[:locale]), T.nilable(Locale))
  end

  sig(:final) { returns(Locale) }
  private def preferred_locale
    preferred_languages = http_accept_language.user_preferred_languages
    # Chrome returns "ja", but Safari would return "ja-JP", not "ja".
    (preferred_languages.any? { |lang| lang.match?(/ja/) }) ? Locale::Ja : default_locale
  end

  sig(:final) { returns(Locale) }
  private def default_locale
    Locale::En
  end

  sig(:final) { returns(Locale) }
  private def current_locale
    instant_locale.presence || current_actor&.locale.presence || preferred_locale
  end
end
