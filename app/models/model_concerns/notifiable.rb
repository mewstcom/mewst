# typed: strict
# frozen_string_literal: true

module ModelConcerns::Notifiable
  extend ActiveSupport::Concern

  included do
    has_one :notification, as: :notifiable, dependent: :restrict_with_exception
  end
end
