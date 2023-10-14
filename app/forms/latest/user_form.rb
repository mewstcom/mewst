# typed: strict
# frozen_string_literal: true

class Latest::UserForm < Latest::ApplicationForm
  attribute :locale, :string

  validates :locale, inclusion: {in: I18n.available_locales.map(&:to_s)}, presence: true
end
